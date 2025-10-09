package push_notifications

import (
	"errors"
	"fmt"

	"github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/firebase"
	repoWebpush "github.com/hngprojects/telex_be/pkg/repository/webpush"
	fcmtokens "github.com/hngprojects/telex_be/services/fcmTokens"
	"github.com/hngprojects/telex_be/utility"
)

func PushFCMToUser(req models.PushRequest, logger *utility.Logger, db *gorm.DB) error {

	title := fmt.Sprintf("Notification from user %s", req.ChannelName)
	body := req.Message

	fcmtoken, exists, err := fcmtokens.GetFcmTokenByUserId(req.UserId, db)

	if !exists {
		return nil
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
		return errors.New(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
	}

	if fcmtoken == "" {
		return nil
	}

	err = firebase.SendNotificationByFCMToken(logger, fcmtoken, title, body, req.AvatarUrl)

	if err != nil {
		logger.Error("Failed to send mass push notification, %s", err.Error())
		return errors.New(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
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
