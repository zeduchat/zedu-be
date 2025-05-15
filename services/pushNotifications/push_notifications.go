package push_notifications

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/firebase"
	fcmtokens "github.com/hngprojects/telex_be/services/fcmTokens"
	"github.com/hngprojects/telex_be/utility"
)

func PushFCMToUser(req models.PushFCMRequest, logger *utility.Logger, db *gorm.DB) error {

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
		logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
		return errors.New(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
	}

	return nil

}

func PushFCMToUsers(req models.PushFCMRequest, logger *utility.Logger, db *gorm.DB) error {

	var (
		channel models.Channels
	)

	userArr := make([]string, 0)

	if len(req.UserIds) == 0 {
		users, err := channel.FetchChannelUsers(db, req.ChannelId)

		if err != nil {
			logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
			return err
		}

		for _, user := range users {
			if user.UserID == req.UserId {
				continue
			}

			userArr = append(userArr, user.UserID)
		}

	} else {
		userArr = req.UserIds
	}

	fcmTokens, err := fcmtokens.GetFcmTokenByUserIds(userArr, db)

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
		return err
	}

	if len(fcmTokens) == 0 {
		return nil
	}

	title := fmt.Sprintf("#%s ", req.ChannelName)
	body := fmt.Sprintf("(@%s): %s", req.Username, req.Message)

	err = firebase.SendNotificationByFCMTokens(logger, fcmTokens, title, body)

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to send mass push notification, %s", err.Error()))
		return fmt.Errorf("Failed to send mass push notification, %s", err.Error())
	}

	return nil
}
