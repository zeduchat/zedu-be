package push_notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/firebase"
	repoWebpush "github.com/hngprojects/telex_be/pkg/repository/webpush"
	fcmtokens "github.com/hngprojects/telex_be/services/fcmTokens"
	notificationpref "github.com/hngprojects/telex_be/services/notification_pref"
	onesignalService "github.com/hngprojects/telex_be/services/onesignal"
	"github.com/hngprojects/telex_be/utility"
)

func PushFCMToUser(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {

	// Check notification preferences if ChannelId and OrgId are provided
	if req.ChannelId != "" && req.OrgId != "" {
		shouldSend, err := notificationpref.ShouldSendNotification(db, req.UserId, req.ChannelId, req.OrgId, notificationpref.NotificationTypeAllMessages)
		if err != nil {
			logger.Error("failed to check notification preferences: %v", err)
			return nil // Non-critical error
		}
		if !shouldSend {
			logger.Info("User %s has muted or disabled notifications for channel %s", req.UserId, req.ChannelId)
			return nil
		}
	}

	return deliverFCM(logger, db, []string{req.UserId}, req.Title, req.Message, req.AvatarUrl, req.Payload, req.OrgId)
}

func PushFCMToUsers(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {

	var (
		channel models.Channels
	)

	userArr := make([]string, 0)

	if len(req.UserIds) == 0 {
		users, err := channel.FetchChannelUsers(db, req.ChannelId, req.UserId)

		if err != nil {
			logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
			return err
		}

		userArr = users

	} else {
		userArr = req.UserIds
	}

	// Filter users by preferences if ChannelId and OrgId provided
	if req.ChannelId != "" && req.OrgId != "" {
		filteredUsers, err := notificationpref.FilterUsersByPreferences(db, userArr, req.ChannelId, req.OrgId, notificationpref.NotificationTypeAllMessages)
		if err != nil {
			logger.Error("failed to filter users by preferences: %v", err)
			return err
		}
		userArr = filteredUsers
	}

	if len(userArr) == 0 {
		logger.Info("No users to send push notification to after preference filtering")
		return nil
	}

	title := ResolvePushTitle(req, db)
	body := ResolvePushBody(req.Username, req.Message)

	return deliverFCM(logger, db, userArr, title, body, req.AvatarUrl, req.Payload, req.OrgId)
}

// SendPush sends a push notification to a user
func SendWebPush(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {

	if req.Payload == "" {
		return nil
	}

	sub, exists, _ := fcmtokens.GetWebPushTokenByUserId(req.UserId, db)

	if !exists {
		return nil
	}

	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.Keys.P256dh,
			Auth:   sub.Keys.Auth,
		},
	}

	err := repoWebpush.SendPush(req.Payload, subscription)

	return err
}

// SendPush sends a push notification to users
func SendWebPushToUsers(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {
	var (
		channel models.Channels
	)

	userArr := make([]string, 0)

	if len(req.UserIds) == 0 {
		users, err := channel.FetchChannelUsers(db, req.ChannelId, req.UserId)

		if err != nil {
			logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
			return err
		}

		userArr = users

	} else {
		userArr = req.UserIds
	}

	webPushTokens, err := fcmtokens.GetWebPushTokenByUserIds(userArr, db)

	if len(*webPushTokens) == 0 {
		return nil
	}

	errs := ""

	for _, sub := range *webPushTokens {

		subscription := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.Keys.P256dh,
				Auth:   sub.Keys.Auth,
			},
		}

		err := repoWebpush.SendPush(req.Payload, subscription)
		errs = fmt.Sprintf("%s///%s", errs, err.Error())

	}

	if errs != "" {
		logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
		return fmt.Errorf("%s", errs)
	}

	return nil
}

func ResolveNotifType(payload any) notificationpref.NotificationType {
	if payloadMap, ok := payload.(map[string]interface{}); ok {
		if notifType, exists := payloadMap["notification_type"].(string); exists {
			switch notifType {
			case "dm":
				return notificationpref.NotificationTypeDirectMessage
			case "channel":
				return notificationpref.NotificationTypeAllMessages
			}
		}
	}
	return notificationpref.NotificationTypeAllMessages
}

// PushOneSignalToUser sends a OneSignal push notification to a single user
func PushOneSignalToUser(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {
	if payloadMap, ok := req.Payload.(map[string]interface{}); ok {
		if msgType, exists := payloadMap["type"].(string); exists && msgType == "system" {
			logger.Info("Skipping OneSignal notification for system message in channel %s", req.ChannelId)
			return nil
		}
	}

	// Check notification preferences if ChannelId and OrgId are provided
	if req.ChannelId != "" && req.OrgId != "" {
		shouldSend, err := notificationpref.ShouldSendNotification(db, req.UserId, req.ChannelId, req.OrgId, ResolveNotifType(req.Payload))
		if err != nil {
			logger.Error("failed to check notification preferences: %v", err)
			return nil // Non-critical error
		}
		if !shouldSend {
			logger.Info("User %s has muted or disabled notifications for channel %s", req.UserId, req.ChannelId)
			return nil
		}
	}

	return deliverFCM(logger, db, []string{req.UserId}, req.Title, req.Message, req.AvatarUrl, req.Payload, req.OrgId)
}

// PushOneSignalToUsers sends a OneSignal push notification to multiple users
func PushOneSignalToUsers(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {
	if payloadMap, ok := req.Payload.(map[string]interface{}); ok {
		if msgType, exists := payloadMap["type"].(string); exists && msgType == "system" {
			logger.Info("Skipping OneSignal notification for system message in channel %s", req.ChannelId)
			return nil
		}
	}
	var (
		channel models.Channels
	)

	userArr := make([]string, 0)

	if len(req.UserIds) == 0 {
		users, err := channel.FetchChannelUsers(db, req.ChannelId, req.UserId)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to fetch channel users: %s", err.Error()))
			return nil
		}
		userArr = users
	} else {
		userArr = req.UserIds
	}

	// Filter users by preferences if ChannelId and OrgId provided and not already filtered
	if !req.SkipPreferenceFilter && req.ChannelId != "" && req.OrgId != "" {
		filteredUsers, err := notificationpref.FilterUsersByPreferences(db, userArr, req.ChannelId, req.OrgId, ResolveNotifType(req.Payload))
		if err != nil {
			logger.Error("failed to filter users by preferences: %v", err)
			return err
		}
		userArr = filteredUsers
	}

	if len(userArr) == 0 {
		logger.Info("No users to send OneSignal notification to after preference filtering")
		return nil
	}

	req.Title = ResolvePushTitle(req, db)
	req.Message = ResolvePushBody(req.Username, req.Message)

	return deliverFCM(logger, db, userArr, req.Title, req.Message, req.AvatarUrl, req.Payload, req.OrgId)
}

// SendOneSignalNotificationToUser sends a OneSignal push notification to a specific user
func SendOneSignalNotificationToUser(logger *utility.Logger, db *gorm.DB, userID string, req models.OneSignalPushRequest) error {
	return deliverFCM(logger, db, []string{userID}, req.Title, req.Message, req.AvatarUrl, nil, "")
}

// PushOneSignalToUsersForBroadcast sends a OneSignal broadcast notification to multiple users
// Used for admin broadcast notifications to all users
func PushOneSignalToUsersForBroadcast(req models.PushRequest, logger *utility.Logger, db *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 {
		logger.Info("No users provided for broadcast notification")
		return nil
	}

	return deliverFCM(logger, db, userIDs, req.Title, req.Message, req.AvatarUrl, req.Payload, req.OrgId)
}

func PushFCMToUserIDs(logger *utility.Logger, db *gorm.DB, userIDs []string, req models.PushRequest) error {
	return deliverFCM(logger, db, userIDs, req.Title, req.Message, req.AvatarUrl, req.Payload, req.OrgId)
}

var protectedRoutingKeys = map[string]bool{
	"channel_id":        true,
	"thread_id":         true,
	"message_id":        true,
	"org_id":            true,
	"sender_id":         true,
	"event":             true,
	"notification_type": true,
	"section":           true,
	"type":              true,
	"buzz_id":           true,
	"buzz_code":         true,
	"host_id":           true,
}

func deliverFCM(logger *utility.Logger, db *gorm.DB, userIDs []string, title, body, avatar string, payload any, orgID string) error {
	tokens, err := models.ListDeviceTokensByUserIDs(db, userIDs)
	if err != nil {
		logger.Error("failed to load device tokens: %v", err)
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	tokenValues := make([]string, 0, len(tokens))
	recipients := make([]string, 0, len(tokens))
	seenUsers := map[string]struct{}{}
	seenTokens := map[string]struct{}{}
	for _, token := range tokens {
		t := strings.TrimSpace(token.Token)
		if t == "" {
			continue
		}
		if _, exists := seenTokens[t]; !exists {
			seenTokens[t] = struct{}{}
			tokenValues = append(tokenValues, t)
		}
		if _, exists := seenUsers[token.UserId]; !exists {
			seenUsers[token.UserId] = struct{}{}
			recipients = append(recipients, token.UserId)
		}
	}
	if len(tokenValues) == 0 {
		return nil
	}

	pushTitle := truncateRunes(strings.TrimSpace(title), 120)
	pushBody := truncateRunes(strings.TrimSpace(body), 240)
	data := payloadToData(payload)
	if content, ok := data["content"]; ok {
		data["content"] = truncateRunes(content, 240)
	}
	data = fitDataPayload(data)

	invalid, successCount, err := firebase.SendPush(logger, tokenValues, pushTitle, pushBody, avatar, data)
	if len(invalid) > 0 {
		if deleteErr := models.DeleteDeviceTokens(db, invalid); deleteErr != nil {
			logger.Error("failed to delete invalid device tokens: %v", deleteErr)
		}
	}
	if successCount == 0 {
		if err != nil {
			logger.Error("failed to send firebase notification: %v", err)
			return err
		}
		return nil
	}
	if err != nil {
		logger.Error("firebase push partially failed: %v", err)
	}

	var payloadMap map[string]interface{}
	if pm, ok := payload.(map[string]interface{}); ok {
		payloadMap = pm
	} else if payload != nil {
		raw, err := json.Marshal(payload)
		if err == nil {
			_ = json.Unmarshal(raw, &payloadMap)
		}
	}
	if _, _, saveErr := onesignalService.SaveBatchNotifications(db, utility.GenerateUUID(), recipients, title, body, avatar, payloadMap, orgID); saveErr != nil {
		logger.Error("failed to save notification history: %v", saveErr)
	}

	return nil
}

func payloadToData(payload any) map[string]string {
	if payload == nil {
		return nil
	}

	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil
		}
		if err := json.Unmarshal(raw, &payloadMap); err != nil || len(payloadMap) == 0 {
			return nil
		}
	}

	data := make(map[string]string, len(payloadMap))
	for key, value := range payloadMap {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		text, ok := dataValue(value)
		if !ok {
			continue
		}
		data[key] = text
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func dataValue(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return "", false
		}
		return text, true
	case map[string]interface{}, []interface{}, map[string]string, []string:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "", false
		}
		text := string(encoded)
		if text == "null" || text == "{}" || text == "[]" {
			return "", false
		}
		return text, true
	default:
		text := strings.TrimSpace(fmt.Sprint(typed))
		if text == "" || text == "<nil>" {
			return "", false
		}
		return text, true
	}
}

func fitDataPayload(data map[string]string) map[string]string {
	if len(data) == 0 {
		return nil
	}

	const maxTotal = 3200
	for dataSize(data) > maxTotal {
		longestKey := ""
		longest := 0
		for key, value := range data {
			if protectedRoutingKeys[key] {
				continue
			}
			if len(value) > longest {
				longest = len(value)
				longestKey = key
			}
		}
		if longestKey == "" {
			break
		}
		if longest <= 32 {
			delete(data, longestKey)
			continue
		}
		next := truncateRunes(data[longestKey], max(longest/2, 32))
		if len(next) >= longest {
			delete(data, longestKey)
			continue
		}
		data[longestKey] = next
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func dataSize(data map[string]string) int {
	size := 0
	for key, value := range data {
		size += len(key) + len(value)
	}
	return size
}

func truncateRunes(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	if maxLen == 1 {
		return string(runes[:1])
	}
	return string(runes[:maxLen-1]) + "…"
}

func ResolvePushTitle(req models.PushRequest, db *gorm.DB) string {
	channelName := strings.TrimSpace(req.ChannelName)

	if channelName == "" && req.ChannelId != "" {
		var ch models.Channels
		if err := db.Where("id = ?", req.ChannelId).Select("name").First(&ch).Error; err == nil && ch.Name != "" {
			channelName = strings.TrimSpace(ch.Name)
		}
	}

	isDM := false
	if payloadMap, ok := req.Payload.(map[string]interface{}); ok {
		if notifType, exists := payloadMap["notification_type"].(string); exists && (notifType == "dm" || notifType == "group_dm") {
			isDM = true
		}
		if section, exists := payloadMap["section"].(string); exists && (section == "dm" || section == "reply_dm") {
			isDM = true
		}
	}

	if channelName != "" {
		if isDM || strings.HasPrefix(channelName, "@") {
			return channelName
		}
		if strings.HasPrefix(channelName, "#") {
			return channelName
		}
		return fmt.Sprintf("#%s", channelName)
	}

	title := strings.TrimSpace(req.Title)
	if title != "" && title != "#" && title != "# " {
		return title
	}
	if req.Username != "" {
		return req.Username
	}
	return "New Message"
}

func ResolvePushBody(username, message string) string {
	msg := strings.TrimSpace(message)
	uname := strings.TrimSpace(username)
	if uname != "" && !strings.HasPrefix(msg, fmt.Sprintf("(%s):", uname)) && !strings.HasPrefix(msg, fmt.Sprintf("(%s)", uname)) {
		return fmt.Sprintf("(%s): %s", uname, msg)
	}
	return msg
}
