package dm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func SaveThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger) (*models.ThreadDocument, int, error) {
	var (
		profile models.Profile
		user    models.User
		channel models.DmChannels
	)

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user")
	}

	exists, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID, req.UserId)
	if !exists || err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
	}

	messageType := "message"
	if req.Type != "" {
		messageType = req.Type
	}
	username := utility.ThisOrThat(profile.UserName, utility.ThisOrThat(profile.FullName, user.Email))

	threadDoc := models.ThreadDocument{
		ID:               utility.GenerateUUID(),
		Username:         profile.UserName,
		Content:          req.Content,
		ChannelsID:       req.ChannelsID,
		Type:             messageType,
		MessageCount:     0,
		AvatarURL:        profile.AvatarURL,
		DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(req.UserId),
		FullName:         profile.FullName,
		Email:            user.Email,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		CurrentStatus:    "pending",
		UserId:           req.UserId,
		Messages:         []models.MessageDocument{},
		Status:           "success",
		ChannelName:      username,
		ChannelType:      "DM",
		Edited:           false,
		UserType:         "user",
		Mentions:         req.Mentions,
		Media:            req.Media,
		OrganisationID:   channel.OrgId,
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create thread: %v", err)
	}

	// Update file metadata with DM channel and message IDs
	if len(req.Media) > 0 {
		fileIDs := make([]string, len(req.Media))
		for i, file := range req.Media {
			fileIDs[i] = file.ID
		}

		// Non-blocking: log error but don't fail DM send
		_ = models.UpdateFilesMetadata(db.Postgresql, logger, fileIDs, req.ChannelsID, threadDoc.ID)
	}

	feed := models.FeedMessageRequest{
		ChannelID:        req.ChannelsID,
		UserName:         profile.UserName,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		AvatarURL:        profile.AvatarURL,
		DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(req.UserId),
		Type:             "message",
		Content:          req.Content,
		ThreadId:         threadDoc.ID,
		Email:            user.Email,
		FullName:         profile.FullName,
		UserId:           req.UserId,
		UserType:         "user",
		Media:            req.Media,
		ChannelName:      username,
		ChannelType:      "DM",
		OrgId:            channel.OrgId,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsID, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, error: %v", req.ChannelsID, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish webhook data: %v", err)
	}

	dataByte, _ := json.Marshal(feed)

	typeChannelId := req.ChannelsID
	channelType := models.GroupDMChannel

	if channel.ChannelType == "dm" {
		typeChannelId = *channel.ParticipantId
		channelType = models.DMChannel
	}

	notifRec := models.PushNotificationRecord{
		ChannelType: channelType,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   typeChannelId,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}

	err = actions.AddPushNotificationToQueue(storage.DB.Redis, notifRec)

	if err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", req.ChannelsID, channel.OrgId, err.Error())
	}

	logger.Info("added notification to queue for channel %s", req.ChannelsID)

	dmChan := models.DmChannels{}
	dmChan.ChannelId = req.ChannelsID
	dmChan.UserId = req.UserId
	dmChan.ChannelType = channel.ChannelType
	dmChan.OrgId = channel.OrgId

	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	// Add to the wait group for each goroutine that must complete first
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Updating unread thread count")
		dmChan.UpdateUnReadCount(db.Postgresql, mutex, logger)
	}()

	// Run this after the other finish
	go func() {
		wg.Wait()
		dmChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread)
	}()

	err = channel.UpdateInteractionAt(db.Postgresql)
	if err != nil {
		logger.Error("Error updating last interacted time of channelid: %s, with orgid: %s error: %v", req.ChannelsID, channel.OrgId, err.Error())
	}

	return &threadDoc, http.StatusCreated, nil
}

func sendDMMessageToBot(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger, extReq request.ExternalRequest) (*models.ThreadDocument, int, error) {
	var (
		profile models.Profile
		user    models.User
		channel models.DmChannels
	)

	err := profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to get user")
	}

	exists, err := channel.CheckChannelExists(db.Postgresql, req.ChannelsID, req.UserId)
	if !exists || err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
	}

	messageType := "message"
	if req.Type != "" {
		messageType = req.Type
	}

	threadDoc := models.ThreadDocument{
		ID:               utility.GenerateUUID(),
		Username:         profile.UserName,
		Content:          req.Content,
		ChannelsID:       req.ChannelsID,
		Type:             messageType,
		MessageCount:     0,
		AvatarURL:        profile.AvatarURL,
		DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(req.UserId),
		FullName:         profile.FullName,
		Email:            user.Email,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		CurrentStatus:    "pending",
		UserId:           req.UserId,
		Messages:         []models.MessageDocument{},
		Status:           "success",
		Edited:           false,
		UserType:         "user",
		Mentions:         req.Mentions,
		Media:            req.Media,
		OrganisationID:   channel.OrgId,
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create thread: %v", err)
	}

	// Update file metadata with DM channel and message IDs
	if len(req.Media) > 0 {
		fileIDs := make([]string, len(req.Media))
		for i, file := range req.Media {
			fileIDs[i] = file.ID
		}

		// Non-blocking: log error but don't fail DM send
		_ = models.UpdateFilesMetadata(db.Postgresql, logger, fileIDs, req.ChannelsID, threadDoc.ID)
	}

	publishfeed := models.FeedMessageRequest{
		ChannelID:        req.ChannelsID,
		UserName:         profile.UserName,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		AvatarURL:        profile.AvatarURL,
		DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(req.UserId),
		Type:             "message",
		Content:          req.Content,
		ThreadId:         threadDoc.ID,
		Email:            user.Email,
		FullName:         profile.FullName,
		UserId:           req.UserId,
		UserType:         "user",
		Media:            req.Media,
	}

	err = centrifuge.PublishChannel(logger, publishfeed.ChannelID, publishfeed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to participant id: %s, error: %v", publishfeed.ChannelID, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish webhook data: %v", err)
	}

	logger.Info(fmt.Sprintf("Publishing to channel id: %s", publishfeed.ChannelID))

	// Process and perform credit validation rules
	inputputLength := len(req.Content)

	var agentPrice float64 = 0.0 // temp value

	creditUsed := models.CalculateCreditCost(inputputLength, agentPrice)

	// validate credit here
	if !models.OrgHasValidCreditBalance(db.Postgresql, channel.OrgId, creditUsed) {
		logger.Error("Organisation has insufficient credit balance!!, orgId: %s", channel.OrgId)
		return nil, http.StatusBadRequest, fmt.Errorf("organisation has insufficient credit balance")
	}

	sendToBot := models.BotRequest{
		ChannelID: req.ChannelsID,
		Content:   req.Content,
		ThreadId:  utility.GenerateUUID(),
		Type:      models.NewBotMessage,
		UserId:    req.UserId,
		OrgId:     req.OrgId,
		Mentions:  req.Mentions,
		Media:     req.Media,
	}

	go func() {
		// Use the injected external request
		_, statusCode, err := BotResponse(sendToBot, db, logger, extReq)
		if err != nil {
			logger.Error(fmt.Sprintf("Error publishing to bot response: %v, statusCode: %d", err, statusCode))
		}
	}()

	return &threadDoc, http.StatusCreated, nil
}

// Direct Message thread
func CreateThreadDmMessage(req models.CreateThreadMsgReq, db *storage.Database, logger *utility.Logger, extReq request.ExternalRequest) (*models.ThreadDocument, int, error) {

	dmchannel := models.DmChannels{}

	exists := postgresql.CheckExists(db.Postgresql, &dmchannel, "channel_id = ? AND chat_type = ?", req.ChannelsID, "bot")
	req.OrgId = dmchannel.OrgId

	if exists {
		return sendDMMessageToBot(req, db, logger, extReq)
	}

	// Create pair room if first message and not a bot
	thread := models.ThreadDocument{
		UserId:     req.UserId,
		ChannelsID: req.ChannelsID,
	}

	pairRoom, code, err := thread.CheckUserThreadExists()
	if err != nil {
		return &thread, code, err
	}

	if !pairRoom {

		dmchannel := models.DmChannels{}

		res, err := dmchannel.CheckChannelExists(db.Postgresql, req.ChannelsID, req.UserId)

		if !res || err != nil {
			return &thread, http.StatusBadRequest, err
		}

		req.OrgId = dmchannel.OrgId

		if dmchannel.ChatType != "bot" && dmchannel.ChannelType == "dm" {

			pairRoomChan := models.DmChannels{}

			pairRoomChan.ChatType = dmchannel.ChatType
			pairRoomChan.ChannelType = "dm"
			pairRoomChan.UserId = *dmchannel.ParticipantId
			pairRoomChan.ParticipantId = &dmchannel.UserId
			pairRoomChan.ID = utility.GenerateUUID()
			pairRoomChan.ChannelId = dmchannel.ChannelId
			pairRoomChan.OrgId = dmchannel.OrgId

			_, err = pairRoomChan.CreateDmChannel(db.Postgresql)
			if err != nil {
				return &thread, http.StatusInternalServerError, err
			}
		}
	}

	return SaveThreadDmMessage(req, db, logger)
}

func GetAllChannelDmThreads(channelID string, db *gorm.DB, c *gin.Context) ([]models.Threads, *elastic.PaginationResponse, int, error) {
	var (
		accessData         models.Threads
		accessResp         []models.Threads
		paginationResponse *elastic.PaginationResponse
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, nil, http.StatusNotFound, err
	}

	userID, ok := userId.(string)
	if !ok {
		return nil, nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := user.GetUser(userID, db)
	if err != nil {
		return nil, nil, code, err
	}

	accessResp, paginationResponse, code, err = accessData.GetAllThreadsByChannelID(c, db, userID, channelID)

	if err != nil {
		return accessResp, nil, code, err
	}

	return accessResp, paginationResponse, http.StatusOK, nil
}

func BotResponse(req models.BotRequest, db *storage.Database, logger *utility.Logger, extReq request.ExternalRequest) (*models.ThreadDocument, int, error) {
	var (
		channel  models.DmChannels
		orgAgent models.OrganisationIntegrations
		// user       models.User
	)

	if req.ThreadId == "00000000-0000-0000-0000-000000000000" {

		var agent models.Integrations

		exists := postgresql.CheckExists(db.Postgresql, &orgAgent, "integration_id = ?", req.AgentId)
		if !exists {

			exists = postgresql.CheckExists(db.Postgresql, &agent, "id = ?", req.AgentId)
			if !exists {
				logger.Info("Failed to get agent info, with marketplace channel id: %v and agent id: %v", req.ChannelID, req.AgentId)
				return nil, http.StatusBadRequest, fmt.Errorf("agent does not exist")
			}
		}

		feed := models.FeedMessageRequest{
			ChannelID:        req.ChannelID,
			UserName:         utility.ThisOrThat(orgAgent.AppName, agent.Name),
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
			AvatarURL:        utility.ThisOrThat(orgAgent.AppLogo, agent.AppLogo),
			DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(req.AgentId),
			Type:             "message",
			Content:          req.Content,
			Email:            "agent",
			FullName:         utility.ThisOrThat(orgAgent.AppName, agent.Name),
			UserType:         "bot",
			State:            req.State,
		}

		err := centrifuge.PublishChannel(logger, req.ChannelID, feed)
		if err != nil {
			logger.Error(fmt.Sprintf("Error Publishing to marketplace channel with destination id: %s error: %v", req.ChannelID, err.Error()))
			return nil, http.StatusBadRequest, errors.New("failed to publish data")
		}

		logger.Info(fmt.Sprintf("Publishing update to marketplace with channel id: %s", req.ChannelID))
		return &models.ThreadDocument{}, http.StatusOK, nil
	}

	exists, err := channel.CheckChannelExists(db.Postgresql, req.ChannelID, "")
	if !exists || err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("channel does not exist: %v", err)
	}

	exists = postgresql.CheckExists(db.Postgresql, &orgAgent, "integration_id = ?", channel.ParticipantId)
	if !exists {
		return nil, http.StatusBadRequest, fmt.Errorf("agent does not exist in org, with integration_id = ?, %v", *channel.ParticipantId)
	}

	if req.Type == models.NewBotMessage {
		fullMessage, err := ProcessBotStreamingResponse(req, orgAgent, channel, extReq, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to process bot streaming response: %v", err))
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to process bot streaming response: %v", err)
		}

		req.Content = fullMessage
	}

	threadDoc := models.ThreadDocument{
		ID:               req.ThreadId,
		Username:         orgAgent.AppName,
		Content:          req.Content,
		ChannelsID:       req.ChannelID,
		Type:             "message",
		MessageCount:     0,
		AvatarURL:        orgAgent.AppLogo,
		DefaultAvatarURL: avatar.GenerateDefaultAvatarURL(*channel.ParticipantId),
		FullName:         orgAgent.AppName,
		Email:            "agent",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		CurrentStatus:    "pending",
		UserId:           *channel.ParticipantId,
		Messages:         []models.MessageDocument{},
		Status:           "success",
		UserType:         "bot",
		Edited:           false,
		Mentions:         req.Mentions,
		Media:            req.Media,
		OrganisationID:   channel.OrgId,
		State:            req.State,
	}

	err = threadDoc.CreateThread(db, logger)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create thread: %v", err)
	}

	feed := models.FeedMessageRequest{
		ChannelID:        req.ChannelID,
		UserName:         orgAgent.AppName,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		AvatarURL:        orgAgent.AppLogo,
		DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(req.AgentId),
		Type:             "message",
		Content:          req.Content,
		ThreadId:         req.ThreadId,
		Email:            "agent",
		FullName:         orgAgent.AppName,
		UserId:           *channel.ParticipantId,
		Media:            req.Media,
		UserType:         "bot",
		State:            req.State,
	}

	notification := models.Notification[models.AgentUpdate]
	notification.SectionType = models.ThreadSection
	notification.Content = feed
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelID,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelID, notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to with destination id: %s error: %v", req.ChannelID, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish data")
	}
	logger.Info("Published complete stream payload channel [%s] for agent: [%s]", req.ChannelID, req.AgentId)

	dmChan := models.DmChannels{}
	dmChan.ChannelId = req.ChannelID
	dmChan.UserId = "00000000-0000-0000-0000-000000000000"
	dmChan.ChannelType = channel.ChannelType
	dmChan.OrgId = channel.OrgId
	dmChan.ChatType = channel.ChatType
	dmChan.ParticipantId = channel.ParticipantId

	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Updating unread thread count")
		dmChan.UpdateUnReadCount(db.Postgresql, mutex, logger)
	}()

	go func() {
		wg.Wait()
		dmChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread)
	}()

	err = channel.UpdateInteractionAt(db.Postgresql)
	if err != nil {
		logger.Error("Error updating last interacted time of channelid: %s, with orgid: %s error: %v", req.ChannelID, channel.OrgId, err.Error())
	}

	logger.Info(fmt.Sprintf("Publishing update to channel id: %s", req.ChannelID))

	pushReq := models.PushRequest{
		ChannelId:   req.ChannelID,
		OrgId:       channel.OrgId,
		ChannelName: feed.UserName,
		UserId:      *channel.ParticipantId,
		Message:     req.Content,
		TimeStamp:   threadDoc.CreatedAt.String(),
		AvatarUrl:   feed.AvatarURL,
		Title:       fmt.Sprintf("Notification from user %s", feed.UserName),
	}

	err = push_notifications.PushOneSignalToUser(pushReq, logger, db.Postgresql)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to send push notification to user %s: %v", *channel.ParticipantId, err))
	}

	creditUsed := models.CalculateCreditCost(len(req.Content), 0)

	if req.OperationPrice != nil && *req.OperationPrice > 0 {
		creditUsed = creditUsed + (*req.OperationPrice) // apply operation price
	}

	credit_usage := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: channel.OrgId,
		Amount:         creditUsed,
		AgentID:        *channel.ParticipantId,
		UserID:         &channel.UserId,
	}

	err = credit_usage.UpdateOrCreateDailyCredit(db.Postgresql, creditUsed)
	if err != nil {
		logger.Error("failed to create/update credit usage!!")
		return nil, http.StatusBadRequest, fmt.Errorf("failed to create/update organisation credit usage: %v", err)
	}

	// create agent bill to an organization only if special operation was performed
	if req.OperationPrice != nil && *req.OperationPrice > 0 {
		err = orgAgent.CreateOrUpdateBillFromUsage(db.Postgresql, &credit_usage)
		if err != nil {
			logger.Error("failed to create/update agent bill")
			return nil, http.StatusBadRequest, fmt.Errorf("failed to create/update agent bill: %v", err)
		}
	}

	// update organization credit balance
	if err = models.UpdateOrgCreditBalance(db.Postgresql, channel.OrgId); err != nil {
		logger.Error("Organisation credit Recalculation failed")
		return nil, http.StatusBadRequest, fmt.Errorf("organisation credit recalculation failed: %v", err)
	}

	// Publish real-time update to superadmin dashboard (async)
	go models.PublishPlatformCreditUpdate(db.Postgresql)

	return &threadDoc, http.StatusCreated, nil
}

func SetupChatSession(req models.UnauthReq, db *storage.Database, logger *utility.Logger) (*models.UserUsageInfoResponse, int, error) {

	unUser := models.UnauthenticatedUser{}

	resp, err := unUser.GetOrCreateUserInfo(db.Postgresql, req.UserIdentifier)

	if err != nil {
		return &models.UserUsageInfoResponse{}, http.StatusInternalServerError, err
	}

	return resp, http.StatusOK, nil

}

func TemporalThreadDmMessage(req models.CreateThreadMsgReq2, db *storage.Database, logger *utility.Logger) (*models.FeedQueue, int, error) {

	var (
		unAuthUser models.UnauthenticatedUser
	)

	agentId, err := models.ResolveAgentId(req.AgentId, db)

	if err != nil {
		logger.Error("Failed to send message to bot for marketplace chat, err: %v", err)
		return &models.FeedQueue{}, http.StatusUnprocessableEntity, err
	}

	req.AgentId = agentId

	resp, code, err := sendTemporalMessageToBot(req, db, logger)

	if err != nil {
		logger.Error("Failed to send message to bot for marketplace chat, err: %v", err)
		return resp, code, errors.New("Failed to send message")
	}

	_, err = unAuthUser.IncrementUsage(db.Postgresql, models.UnauthReq{
		UserIdentifier:  req.UserIdentifier,
		LastAgentChatID: req.AgentId,
		Limit:           models.CHATUSAGELIMIT,
		ChannelID:       req.ChannelsID,
	})

	if err != nil {
		logger.Error("******** Failed to update user usage, err: %v", err)
	}

	return resp, code, err
}

func sendTemporalMessageToBot(req models.CreateThreadMsgReq2, db *storage.Database, logger *utility.Logger) (*models.FeedQueue, int, error) {
	var (
		profile     models.Profile
		user        models.User
		channel     models.DmChannels
		routing_key = "direct_message"
	)

	publishfeed := models.FeedMessageRequest{
		ChannelID:        req.ChannelsID,
		UserName:         profile.UserName,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(req.UserIdentifier),
		Type:             "message",
		Content:          req.Content,
		Email:            user.Email,
		FullName:         profile.FullName,
		UserType:         "user",
		ThreadId:         "00000000-0000-0000-0000-000000000000",
	}

	err := centrifuge.PublishChannel(logger, publishfeed.ChannelID, publishfeed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to participant id: %s, error: %v", publishfeed.ChannelID, err))
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to publish feed data: %v", err)
	}

	logger.Info(fmt.Sprintf("Publishing to marketplace channel with id: %s", publishfeed.ChannelID))

	returnUrl := fmt.Sprintf("%s/api/v1/dms/bot-dm-response", config.Config.App.Url)
	feed := models.FeedQueue{
		ChannelsId: req.ChannelsID,
		Content:    req.Content,
		ThreadId:   "00000000-0000-0000-0000-000000000000",
		ReturnUrl:  returnUrl,
		Type:       "message/thread",
		UserId:     "00000000-0000-0000-0000-000000000000",
		OrgId:      "00000000-0000-0000-0000-000000000000",
		AgentId:    req.AgentId,
	}

	payload := map[string]any{
		"message_content": map[string]any{
			"channel_id":              feed.ChannelsId,
			"message":                 feed.Content,
			"thread_id":               feed.ThreadId,
			"is_channel_conversation": false,
			"type":                    feed.Type,
			"user_id":                 feed.UserId,
			"org_id":                  feed.OrgId,
			"media":                   feed.Media,
			"mentions":                feed.Mentions,
			"agent_id":                feed.AgentId,
		},
		"channel_id": feed.ChannelsId,
		"org_id":     feed.OrgId,
		"return_url": feed.ReturnUrl,
		"agent_id":   channel.ParticipantId,
	}

	task := "telex_queue_processor.handle_direct_message"

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshaling payload for integration: %v", err.Error()))
		return &models.FeedQueue{}, http.StatusInternalServerError, fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db.Postgresql, string(payloadBytes), routing_key, task)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
		return &models.FeedQueue{}, http.StatusInternalServerError, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	logger.Info("Pushed to RabbitMQ for integration: %s", routing_key)

	return &feed, http.StatusCreated, nil
}
