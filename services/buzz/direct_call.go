package buzz

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/apns"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/onesignal"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"github.com/lib/pq"
)

func InitiateDirectCall(db *storage.Database, logger *utility.Logger, channelID, callerID string) (models.DirectCallResponse, int, error) {
	var resp models.DirectCallResponse

	dmType, err := models.GetDMChannelType(db.Postgresql, channelID)
	if err != nil {
		logger.Error("direct call: channel not found %s: %v", channelID, err)
		return resp, http.StatusNotFound, errors.New("channel not found")
	}

	if dmType != "dm" && dmType != "group_dm" {
		return resp, http.StatusBadRequest, errors.New("direct calls are only supported in DM or group DM channels")
	}

	if !models.IsUserInChannel(db.Postgresql, channelID, callerID) {
		return resp, http.StatusForbidden, errors.New("you are not a member of this channel")
	}

	if err := permissions.CanCreateBuzz(db.Postgresql, channelID, callerID); err != nil {
		statusCode, errMsg := mapPermissionError(err, "create direct call")
		return resp, statusCode, errors.New(errMsg)
	}

	service := agora.Client.Service
	if service == nil {
		logger.Error(errorAgoraNotInitialized)
		return resp, http.StatusInternalServerError, errors.New(errorAgoraNotInitialized)
	}

	channelType := models.ChannelTypeDM
	if dmType == "group_dm" {
		channelType = models.ChannelTypeGroupDM
	}

	participantIDs, err := models.GetDMParticipants(db.Postgresql, channelID)
	if err != nil {
		logger.Error("direct call: failed to fetch channel participants: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch channel participants")
	}

	now := time.Now().UTC()
	endTime := now.Add(time.Duration(DefaultBuzzDurationMinutes) * time.Minute)

	buzz := models.Buzz{
		ID:             utility.GenerateUUID(),
		ChannelID:      channelID,
		ChannelType:    channelType,
		HostID:         callerID,
		OriginalHostID: callerID,
		ParticipantIDs: pq.StringArray(participantIDs),
		BuzzStartTime:  now,
		BuzzEndTime:    &endTime,
		IsLiveStatus:   true,
		Status:         models.BuzzStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	token, err := service.GenerateRTCToken(buzz.ID, callerID, callerID, agora.DefaultTokenExpirationSeconds)
	if err != nil {
		logger.Error("direct call: agora token generation failed: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to generate access token")
	}

	agoraToken := models.BuzzAgoraTokenResponse{
		Token:       token,
		AppId:       service.GetAppId(),
		ChannelName: buzz.ID,
		UID:         callerID,
	}

	err = db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&buzz).Error; err != nil {
			return err
		}
		for _, uid := range participantIDs {
			status := models.CallStatusPending
			if uid == callerID {
				status = models.CallStatusAccepted
			}
			p := models.BuzzParticipant{
				ID:       utility.GenerateUUID(),
				BuzzID:   buzz.ID,
				UserID:   uid,
				Status:   status,
				IsMuted:  false,
				JoinedAt: now,
			}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("direct call: transaction failed: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to create direct call")
	}

	callerName := resolveUsername(db.Postgresql, logger, callerID)

	callParticipants := buildCallParticipants(db.Postgresql, logger, participantIDs, callerID)

	notification := models.Notification[models.DirectCallInitiated]
	notification.SectionType = models.ChannelsSection
	notification.Content = models.DirectCallCentrifugoPayload{
		Event:        string(models.DirectCallInitiated),
		BuzzID:       buzz.ID,
		ChannelID:    channelID,
		CallerID:     callerID,
		CallerName:   callerName,
		Participants: callParticipants,
		CreatedAt:    now,
	}

	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: channelID,
	}

	notification.NotificationId = utility.GenerateUUID()

	var orgID string
	db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ?", channelID).Select("org_id").Limit(1).Scan(&orgID)

	if orgID != "" {
		broadcastChannels := make([]string, 0, len(participantIDs))
		for _, uid := range participantIDs {
			broadcastChannels = append(broadcastChannels, fmt.Sprintf("%s/%s", orgID, uid))
		}

		if err := centrifuge.BatchBroadcastToChannel(logger, broadcastChannels, notification); err != nil {
			logger.Error("direct call: failed to broadcast centrifugo event: %v", err)
		}
	}

	go notifyCallParticipants(db, logger, participantIDs, callerID, callerName, channelID, buzz.ID)

	resp = models.DirectCallResponse{
		BuzzID:       buzz.ID,
		BuzzCode:     utility.ExtractBuzzCode(buzz.ID),
		ChannelID:    channelID,
		CallerID:     callerID,
		CallerName:   callerName,
		Status:       models.BuzzStatusActive,
		Participants: callParticipants,
		CreatedAt:    now,
		AgoraToken:   &agoraToken,
	}

	logger.Info("direct call initiated by %s in channel %s, buzz %s", callerID, channelID, buzz.ID)
	return resp, http.StatusCreated, nil
}

func CancelDirectCall(db *storage.Database, logger *utility.Logger, buzzID, userID string) (models.DirectCallResponse, int, error) {
	var resp models.DirectCallResponse

	var buzz models.Buzz
	if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("call not found")
		}
		return resp, http.StatusInternalServerError, errors.New("failed to fetch call")
	}

	if buzz.Status != models.BuzzStatusActive {
		return resp, http.StatusConflict, errors.New("call has already ended")
	}

	now := time.Now().UTC()
	if err := db.Postgresql.Model(&buzz).Updates(map[string]interface{}{
		"status":         models.BuzzStatusEnded,
		"is_live_status": false,
		"buzz_end_time":  &now,
		"updated_at":     now,
	}).Error; err != nil {
		logger.Error("CancelDirectCall: failed to update buzz status: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to cancel call")
	}

	if err := db.Postgresql.Model(&models.BuzzParticipant{}).
		Where("buzz_id = ?", buzz.ID).
		Update("status", models.BuzzParticipantStatusLeft).Error; err != nil {
		logger.Error("CancelDirectCall: failed to update participant status: %v", err)
	}

	if err := models.ExpireInvitationsForBuzz(db.Postgresql, buzzID); err != nil {
		logger.Error("CancelDirectCall: failed to expire invitations: %v", err)
	}

	participantIDs, _ := models.GetDMParticipants(db.Postgresql, buzz.ChannelID)
	callerName, avatarURL := resolveUserProfile(db.Postgresql, logger, buzz.HostID)

	others := make([]string, 0, len(participantIDs)-1)
	for _, uid := range participantIDs {
		if uid != buzz.HostID {
			others = append(others, uid)
		}
	}

	if len(others) > 0 {
		req := models.PushRequest{
			Title:   "Missed Call",
			Message: fmt.Sprintf("Missed call from %s", callerName),
			UserIds: others,
		}

		pushMsg := models.CallPushPayload{
			BuzzID:           buzz.ID,
			ChannelID:        buzz.ChannelID,
			CallerName:       callerName,
			CallerID:         buzz.HostID,
			AvatarURL:        avatarURL,
			DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(buzz.HostID),
			Event:            string(models.DirectCallCanceled),
		}

		sendDirectCallOneSignal(db, logger, others, req, pushMsg)
		go sendDirectCallVoIP(db, logger, others, req, pushMsg)
	}

	respNotification := models.Notification[models.DirectCallCanceled]
	respNotification.SectionType = models.ChannelsSection
	respNotification.NotificationId = utility.GenerateUUID()
	respNotification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	respNotification.Content = models.DirectCallCentrifugoPayload{
		Event:      string(models.DirectCallCanceled),
		BuzzID:     buzzID,
		ChannelID:  buzz.ChannelID,
		CallerID:   buzz.HostID,
		CallerName: callerName,
		CreatedAt:  buzz.CreatedAt,
	}

	var orgID string
	db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ?", buzz.ChannelID).Select("org_id").Limit(1).Scan(&orgID)

	if orgID != "" {
		broadcastChannels := make([]string, 0, len(participantIDs))
		for _, uid := range participantIDs {
			broadcastChannels = append(broadcastChannels, fmt.Sprintf("%s/%s", orgID, uid))
		}
		if err := centrifuge.BatchBroadcastToChannel(logger, broadcastChannels, respNotification); err != nil {
			logger.Error("CancelDirectCall: failed to broadcast centrifugo cancel event: %v", err)
		}
	}

	resp = models.DirectCallResponse{
		BuzzID:     buzzID,
		BuzzCode:   utility.ExtractBuzzCode(buzzID),
		ChannelID:  buzz.ChannelID,
		CallerID:   buzz.HostID,
		CallerName: callerName,
		Status:     models.BuzzStatusEnded,
		JoinStatus: "canceled",
		CreatedAt:  buzz.CreatedAt,
	}

	return resp, http.StatusOK, nil
}

func RespondToCall(db *storage.Database, logger *utility.Logger, buzzID, userID, action string) (models.DirectCallResponse, int, error) {
	var resp models.DirectCallResponse

	var buzz models.Buzz
	if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("call not found")
		}
		return resp, http.StatusInternalServerError, errors.New("failed to fetch call")
	}

	if buzz.Status != models.BuzzStatusActive {
		return resp, http.StatusConflict, errors.New("call has already ended")
	}

	if action == "cancel" {
		return CancelDirectCall(db, logger, buzzID, userID)
	}

	var participant models.BuzzParticipant
	if err := db.Postgresql.Where("buzz_id = ? AND user_id = ?", buzzID, userID).First(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("respond to call: participant not found for user: %s in buzz: %s", userID, buzzID)
			return resp, http.StatusForbidden, errors.New("you are not a participant in this call")
		}
		return resp, http.StatusInternalServerError, errors.New("failed to fetch participant")
	}

	resolvedAction := action
	ringTimeout := time.Duration(models.DirectCallRingingTimeoutMinutes) * time.Minute
	if time.Since(buzz.CreatedAt) > ringTimeout {
		resolvedAction = models.CallStatusTimeout
	}

	newStatus := actionToStatus(resolvedAction)
	now := time.Now().UTC()

	updates := map[string]interface{}{
		"status":     newStatus,
		"updated_at": now,
	}
	if newStatus == models.CallStatusAccepted {
		updates["joined_at"] = now
	}

	if err := db.Postgresql.Model(&participant).Updates(updates).Error; err != nil {
		logger.Error("respond to call: failed to update participant status: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to update call status")
	}

	var agoraToken *models.BuzzAgoraTokenResponse
	if newStatus == models.CallStatusAccepted {
		service := agora.Client.Service
		if service != nil {
			remaining := buzz.GetRemainingTime(agora.DefaultTokenExpirationSeconds)
			if buzz.BuzzType == models.BuzzTypeOrganization {
				remaining = agora.DefaultTokenExpirationSeconds
			}
			if remaining > 0 {
				token, err := service.GenerateRTCToken(buzzID, userID, userID, remaining)
				if err != nil {
					logger.Error("respond to call: agora token generation failed: %v", err)
				} else {
					agoraToken = &models.BuzzAgoraTokenResponse{
						Token:       token,
						AppId:       service.GetAppId(),
						ChannelName: buzzID,
						UID:         userID,
					}

					if err := buzz.AppendParticipant(db.Postgresql, userID); err != nil {
						logger.Error("respond to call: failed to add user to participants array: %v", err)
					}
				}
			}
		}
	}

	allParticipants := fetchCallParticipants(db.Postgresql, logger, buzzID)

	var respondingUser *models.DirectCallParticipant
	for i := range allParticipants {
		if allParticipants[i].UserID == userID {
			respondingUser = &allParticipants[i]
			break
		}
	}

	callerName := resolveUsername(db.Postgresql, logger, buzz.HostID)

	respNotification := models.Notification[models.DirectCallResponseEvent]
	respNotification.SectionType = models.ChannelsSection
	payload := models.DirectCallCentrifugoPayload{
		Event:        string(models.DirectCallResponseEvent),
		BuzzID:       buzzID,
		ChannelID:    buzz.ChannelID,
		CallerID:     buzz.HostID,
		CallerName:   callerName,
		JoinStatus:   resolvedAction,
		Participants: allParticipants,
		CreatedAt:    buzz.CreatedAt,
	}

	switch newStatus {
	case models.CallStatusAccepted:
		payload.UserJoined = respondingUser
	case models.CallStatusDeclined:
		payload.UserRejected = respondingUser
	case models.CallStatusTimeout:
		payload.UserTimeout = respondingUser
	}

	respNotification.Content = payload
	respNotification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	respNotification.NotificationId = utility.GenerateUUID()

	var orgID string
	db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ?", buzz.ChannelID).Select("org_id").Limit(1).Scan(&orgID)

	if orgID != "" {
		participantIDs, err := models.GetDMParticipants(db.Postgresql, buzz.ChannelID)
		if err == nil {
			broadcastChannels := make([]string, 0, len(participantIDs))
			for _, uid := range participantIDs {
				broadcastChannels = append(broadcastChannels, fmt.Sprintf("%s/%s", orgID, uid))
			}
			if err := centrifuge.BatchBroadcastToChannel(logger, broadcastChannels, respNotification); err != nil {
				logger.Error("respond to call: failed to broadcast centrifugo event: %v", err)
			}
		} else {
			logger.Error("respond to call: failed to fetch channel participants for broadcast: %v", err)
		}
	}

	resp = models.DirectCallResponse{
		BuzzID:       buzzID,
		BuzzCode:     utility.ExtractBuzzCode(buzzID),
		ChannelID:    buzz.ChannelID,
		CallerID:     buzz.HostID,
		CallerName:   callerName,
		Status:       buzz.Status,
		JoinStatus:   resolvedAction,
		Participants: allParticipants,
		CreatedAt:    buzz.CreatedAt,
		AgoraToken:   agoraToken,
	}

	switch newStatus {
	case models.CallStatusAccepted:
		resp.UserJoined = respondingUser
	case models.CallStatusDeclined:
		resp.UserRejected = respondingUser
	case models.CallStatusTimeout:
		resp.UserTimeout = respondingUser
	}

	logger.Info("user %s responded to call %s with action: %s (resolved: %s)", userID, buzzID, action, newStatus)
	return resp, http.StatusOK, nil
}

func actionToStatus(action string) string {
	switch action {
	case "accept":
		return models.CallStatusAccepted
	case "decline":
		return models.CallStatusDeclined
	default:
		return models.CallStatusTimeout
	}
}

func resolveUsername(db *gorm.DB, logger *utility.Logger, userID string) string {
	var profile models.Profile
	if err := db.Where("userid = ?", userID).First(&profile).Error; err != nil {
		logger.Error("resolveUsername: failed to fetch profile for %s: %v", userID, err)
		return ""
	}
	if profile.UserName != "" {
		return profile.UserName
	}
	var user models.User
	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return ""
	}
	if idx := strings.Index(user.Email, "@"); idx != -1 {
		return user.Email[:idx]
	}
	return user.Email
}

func resolveUserProfile(db *gorm.DB, logger *utility.Logger, userID string) (username, avatarURL string) {
	var profile models.Profile
	if err := db.Where("userid = ?", userID).First(&profile).Error; err != nil {
		logger.Error("resolveUserProfile: failed to fetch profile for %s: %v", userID, err)
		return resolveUsername(db, logger, userID), ""
	}
	name := profile.UserName
	if name == "" {
		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err == nil {
			if idx := strings.Index(user.Email, "@"); idx != -1 {
				name = user.Email[:idx]
			} else {
				name = user.Email
			}
		}
	}
	return name, profile.AvatarURL
}

func buildCallParticipants(db *gorm.DB, logger *utility.Logger, participantIDs []string, callerID string) []models.DirectCallParticipant {
	result := make([]models.DirectCallParticipant, 0, len(participantIDs))
	for _, uid := range participantIDs {
		joinStatus := models.CallStatusPending
		if uid == callerID {
			joinStatus = models.CallStatusAccepted
		}
		username, avatarURL := resolveUserProfile(db, logger, uid)
		
		callRole := "receiver"
		if uid == callerID {
			callRole = "caller"
		}

		result = append(result, models.DirectCallParticipant{
			UserID:           uid,
			Username:         username,
			AvatarURL:        avatarURL,
			DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(uid),
			JoinStatus:       joinStatus,
			Color:            utility.GenerateUserColor(uid, username),
			CallRole:         callRole,
		})
	}
	return result
}

func fetchCallParticipants(db *gorm.DB, logger *utility.Logger, buzzID string) []models.DirectCallParticipant {
	var participants []models.BuzzParticipant
	if err := db.Where("buzz_id = ?", buzzID).Find(&participants).Error; err != nil {
		logger.Error("fetchCallParticipants: failed to fetch: %v", err)
		return nil
	}

	var hostID string
	var b models.Buzz
	if err := db.Model(&models.Buzz{}).Select("host_id").Where("id = ?", buzzID).First(&b).Error; err == nil {
		hostID = b.HostID
	}

	result := make([]models.DirectCallParticipant, 0, len(participants))
	for _, p := range participants {
		username, avatarURL := resolveUserProfile(db, logger, p.UserID)
		
		callRole := "receiver"
		if hostID != "" && p.UserID == hostID {
			callRole = "caller"
		}

		result = append(result, models.DirectCallParticipant{
			UserID:           p.UserID,
			Username:         username,
			AvatarURL:        avatarURL,
			DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(p.UserID),
			JoinStatus:       p.Status,
			Color:            utility.GenerateUserColor(p.UserID, username),
			CallRole:         callRole,
		})
	}
	return result
}

func notifyCallParticipants(db *storage.Database, logger *utility.Logger, participantIDs []string, callerID, callerName, channelID, buzzID string) {
	others := make([]string, 0, len(participantIDs)-1)
	for _, uid := range participantIDs {
		if uid != callerID {
			others = append(others, uid)
		}
	}
	if len(others) == 0 {
		return
	}

	callerName, avatarURL := resolveUserProfile(db.Postgresql, logger, callerID)
	pushMsg := models.CallPushPayload{
		BuzzID:           buzzID,
		ChannelID:        channelID,
		CallerName:       callerName,
		CallerID:         callerID,
		AvatarURL:        avatarURL,
		DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(callerID),
		Event:            string(models.DirectCallInitiated),
	}

	req := models.PushRequest{
		Title:   "Incoming Call",
		Message: fmt.Sprintf("You have a new call from %s", callerName),
		UserIds: others,
	}

	sendDirectCallOneSignal(db, logger, others, req, pushMsg)
	go sendDirectCallVoIP(db, logger, others, req, pushMsg)
}

func notifyCallCanceled(db *storage.Database, logger *utility.Logger, participantIDs []string, callerID, callerName, channelID, buzzID string) {
	others := make([]string, 0, len(participantIDs)-1)
	for _, uid := range participantIDs {
		if uid != callerID {
			others = append(others, uid)
		}
	}
	if len(others) == 0 {
		return
	}

	callerName, avatarURL := resolveUserProfile(db.Postgresql, logger, callerID)
	pushMsg := models.CallPushPayload{
		BuzzID:           buzzID,
		ChannelID:        channelID,
		CallerName:       callerName,
		CallerID:         callerID,
		AvatarURL:        avatarURL,
		DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(callerID),
		Event:            string(models.DirectCallCanceled),
	}

	req := models.PushRequest{
		Title:   "Call Canceled",
		Message: fmt.Sprintf("%s has canceled the call", callerName),
		UserIds: others,
	}

	sendDirectCallOneSignal(db, logger, others, req, pushMsg)
}

func handleDirectCallCancellation(db *storage.Database, logger *utility.Logger, buzz models.Buzz, buzzID string) (models.DirectCallResponse, int, error) {
	participantIDs, _ := models.GetDMParticipants(db.Postgresql, buzz.ChannelID)
	callerName := resolveUsername(db.Postgresql, logger, buzz.HostID)

	// notify onesignal
	go notifyCallCanceled(db, logger, participantIDs, buzz.HostID, callerName, buzz.ChannelID, buzz.ID)

	// notify centrifugo
	respNotification := models.Notification[models.DirectCallCanceled]
	respNotification.SectionType = models.ChannelsSection
	respNotification.NotificationId = utility.GenerateUUID()
	respNotification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	respNotification.Content = models.DirectCallCentrifugoPayload{
		Event:      string(models.DirectCallCanceled),
		BuzzID:     buzzID,
		ChannelID:  buzz.ChannelID,
		CallerID:   buzz.HostID,
		CallerName: callerName,
		CreatedAt:  buzz.CreatedAt,
	}

	var orgID string
	db.Postgresql.Model(&models.DmChannels{}).Where("channel_id = ?", buzz.ChannelID).Select("org_id").Limit(1).Scan(&orgID)

	if orgID != "" {
		broadcastChannels := make([]string, 0, len(participantIDs))
		for _, uid := range participantIDs {
			broadcastChannels = append(broadcastChannels, fmt.Sprintf("%s/%s", orgID, uid))
		}
		if err := centrifuge.BatchBroadcastToChannel(logger, broadcastChannels, respNotification); err != nil {
			logger.Error("respond to call: failed to broadcast centrifugo cancel event: %v", err)
		}
	}

	resp := models.DirectCallResponse{
		BuzzID:     buzzID,
		BuzzCode:   utility.ExtractBuzzCode(buzzID),
		ChannelID:  buzz.ChannelID,
		CallerID:   buzz.HostID,
		CallerName: callerName,
		Status:     models.BuzzStatusEnded,
		CreatedAt:  buzz.CreatedAt,
	}

	return resp, http.StatusOK, nil
}

func sendDirectCallOneSignal(db *storage.Database, logger *utility.Logger, userIDs []string, req models.PushRequest, payload models.CallPushPayload) {
	var users []models.User
	if err := db.Postgresql.Where("id IN ?", userIDs).
		Select("id", "onesignal_subscription_id").
		Find(&users).Error; err != nil {
		logger.Error("sendDirectCallOneSignal: failed to fetch users: %v", err)
		return
	}

	subscriptionIDs := make([]string, 0, len(users))
	for _, u := range users {
		if u.OneSignalSubscriptionID != "" {
			subscriptionIDs = append(subscriptionIDs, u.OneSignalSubscriptionID)
		}
	}

	if len(subscriptionIDs) == 0 {
		return
	}

	callData := map[string]interface{}{
		"buzz_id":            payload.BuzzID,
		"channel_id":         payload.ChannelID,
		"caller_name":        payload.CallerName,
		"caller_id":          payload.CallerID,
		"avatar_url":         payload.AvatarURL,
		"default_avatar_url": payload.DefaultAvatarURL,
		"event":              payload.Event,
	}

	if err := onesignal.SendDirectCallNotification(logger, subscriptionIDs, req, callData, db.Postgresql, userIDs); err != nil {
		logger.Error("sendDirectCallOneSignal: failed to send notification: %v", err)
	}
}

func sendDirectCallVoIP(db *storage.Database, logger *utility.Logger, userIDs []string, req models.PushRequest, payload models.CallPushPayload) {
	var tokens []models.FcmTokens
	if err := db.Postgresql.Where("user_id IN ? AND voip_token IS NOT NULL AND voip_token != ''", userIDs).
		Find(&tokens).Error; err != nil {
		logger.Error("sendDirectCallVoIP: failed to fetch VoIP tokens: %v", err)
		return
	}

	voipTokens := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t.VoIPToken != "" {
			voipTokens = append(voipTokens, t.VoIPToken)
		}
	}

	if len(voipTokens) == 0 {
		return
	}

	callData := map[string]interface{}{
		"buzz_id":            payload.BuzzID,
		"channel_id":         payload.ChannelID,
		"caller_name":        payload.CallerName,
		"caller_id":          payload.CallerID,
		"avatar_url":         payload.AvatarURL,
		"default_avatar_url": payload.DefaultAvatarURL,
		"event":              payload.Event,
	}

	if err := apns.SendDirectCallVoIPNotification(logger, voipTokens, req, callData, db.Postgresql, userIDs); err != nil {
		logger.Error("sendDirectCallVoIP: failed to send notification: %v", err)
	}
}

func sendDirectCallCancelVoIP(db *storage.Database, logger *utility.Logger, userIDs []string, payload models.CallPushPayload, buzzID string) {
	req := models.PushRequest{
		Title:   "Call Canceled",
		Message: fmt.Sprintf("%s has canceled the call", payload.CallerName),
		UserIds: userIDs,
	}
	sendDirectCallVoIP(db, logger, userIDs, req, payload)
}
