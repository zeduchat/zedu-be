package push_notifications

import (
	"fmt"

	"github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/firebase"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/onesignal"
	repoWebpush "github.com/hngprojects/telex_be/pkg/repository/webpush"
	fcmtokens "github.com/hngprojects/telex_be/services/fcmTokens"
	"github.com/hngprojects/telex_be/utility"
)

func PushFCMToUser(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {

	title := req.Title
	body := req.Message

	fcmtoken, exists, err := fcmtokens.GetFcmTokenByUserId(req.UserId, db)

	if !exists {
		return nil
	}

	if err != nil {
		logger.Error(fmt.Sprintf("failed to send mass push notification, %s", err.Error()))
		return fmt.Errorf("failed to send mass push notification, %s", err.Error())
	}

	if fcmtoken == "" {
		return nil
	}

	err = firebase.SendNotificationByFCMToken(logger, fcmtoken, title, body, req.AvatarUrl)

	if err != nil {
		logger.Error("Failed to send mass push notification, %s", err.Error())
		return fmt.Errorf("Failed to send mass push notification, %s", err.Error())
	}

	return nil
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

	fcmTokens, err := fcmtokens.GetFcmTokenByUserIds(userArr, db)

	if len(*fcmTokens) == 0 {
		return nil
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
		return err
	}

	title := fmt.Sprintf("#%s ", req.ChannelName)
	body := fmt.Sprintf("(@%s): %s", req.Username, req.Message)

	err = firebase.SendNotificationByFCMTokens(logger, *fcmTokens, title, body)

	if err != nil {
		logger.Error("Failed to send mass push notification, %s", err.Error())
		return fmt.Errorf("Failed to send mass push notification, %s", err.Error())
	}

	return nil
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

// PushOneSignalToUser sends a OneSignal push notification to a single user
func PushOneSignalToUser(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {
	user := &models.User{}

	subscriptionID, err := user.GetOneSignalSubscriptionID(db, req.UserId)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get OneSignal subscription ID: %s", err.Error()))
		return nil
	}

	if subscriptionID == "" {
		return nil
	}

	err = onesignal.OptionalSendNotification(logger, subscriptionID, req)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to send OneSignal notification: %s", err.Error()))
		return nil // Non-critical error, don't fail the notification flow
	}

	return nil
}

// PushOneSignalToUsers sends a OneSignal push notification to multiple users
func PushOneSignalToUsers(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {
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

	// OPTIMIZATION: Single query instead of N+1 (matching FCM pattern at push_notifications.go:69)
	var users []models.User
	if err := db.Where("id IN ?", userArr).
		Select("id", "onesignal_subscription_id").
		Find(&users).Error; err != nil {
		logger.Error(fmt.Sprintf("failed to fetch users: %s", err.Error()))
		return nil
	}

	// Filter out empty subscription IDs
	subscriptionIDs := make([]string, 0, len(users))
	for _, user := range users {
		if user.OneSignalSubscriptionID != "" {
			subscriptionIDs = append(subscriptionIDs, user.OneSignalSubscriptionID)
		}
	}

	if len(subscriptionIDs) == 0 {
		return nil
	}

	req.Title = fmt.Sprintf("#%s ", req.ChannelName)
	req.Message = fmt.Sprintf("(@%s): %s", req.Username, req.Message)

	err := onesignal.OptionalSendBatchNotifications(logger, subscriptionIDs, req)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to send batch OneSignal notification: %s", err.Error()))
		return nil // Non-critical error, don't fail the notification flow
	}

	return nil
}
