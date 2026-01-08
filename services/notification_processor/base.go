package notification_processor

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/utility"
)

func ProcessNotification(req Job, logger *utility.Logger) error {

	var (
		feed = models.FeedMessageRequest{}
		db   = storage.DB.Postgresql
	)

	notification := req.Notification

	err := json.Unmarshal([]byte(notification.Data), &feed)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}

	notifData := models.Notification[notification.Type]
	notifData.SectionType = notification.Section
	notifData.Content = feed
	notifData.UpdateChange = notification.UpdateChange

	processPayload := models.NotificationProcessPayload{
		Notification: notifData,
		OrgId:        feed.OrgId,
		ChannelId:    notification.ChannelId,
		UserId:       feed.UserId,
		ChannelType:  notification.ChannelType,
	}

	typeCall := map[models.ChannelType]func(db *gorm.DB, notification models.NotificationProcessPayload, logger *utility.Logger) error{
		models.Channel:        ChannelNotification,
		models.DMChannel:      DMNotification,
		models.GroupDMChannel: DMNotification,
	}

	err = typeCall[notification.ChannelType](db, processPayload, logger)

	if err != nil {
		return fmt.Errorf("error sending notification data, %v", err)
	}

	return nil
}

func ChannelNotification(db *gorm.DB, notifPayload models.NotificationProcessPayload, logger *utility.Logger) error {

	var (
		channelId = notifPayload.ChannelId
		orgId     = notifPayload.OrgId
		userId    = notifPayload.UserId
		userIDs   []string
	)

	err := db.
		Model(&models.UserChannels{}).
		Where("channels_id = ? AND user_id != ?", channelId, userId).
		Where(`preferences->'web'->>'muted' = ? OR preferences = '{}' OR preferences->'mobile'->>'muted' = ?`, "false", "false").
		Pluck("user_id", &userIDs).Error

	if err != nil {
		return fmt.Errorf("failed to query entry of userids")
	}

	orgUserIds := make([]string, 0)

	for _, userId := range userIDs {
		orgUserIds = append(orgUserIds, fmt.Sprintf("%s/%s", orgId, userId))
	}
	notifPayload.Notification.NotificationId = utility.GenerateUUID()

	err = centrifuge.BatchBroadcastToChannel(logger, orgUserIds, notifPayload.Notification)
	if err != nil {
		logger.Error("Error Publishing to channelid: %s, with orgid: %s error: %v", channelId, orgId, err.Error())
		return fmt.Errorf("failed to publish thread data")
	}

	logger.Info("published new_message notification to %d users", len(userIDs))

	// Push fcm notification to channel users

	feed := notifPayload.Notification.Content.(models.FeedMessageRequest)

	pushReq := models.PushRequest{
		ChannelId:   channelId,
		ChannelName: feed.ChannelName,
		UserIds:     userIDs,
		Message:     feed.Content,
		UserId:      userId,
		Username:    utility.ThisOrThat(feed.UserName, strings.Split(feed.Email, "@")[0]),
		Title:       fmt.Sprintf("Notification from user %s", feed.ChannelName),
	}

	err = push_notifications.PushFCMToUsers(pushReq, logger, db)
	if err != nil {
		logger.Error("failed to send push notifcation to channel users, Err: %v", err.Error())
	}

	logger.Info("sent fcm push notification to channel users")

	// Push OneSignal notification to channel users
	err = push_notifications.PushOneSignalToUsers(pushReq, logger, db)
	if err != nil {
		logger.Error("failed to send OneSignal notification to channel users, Err: %v", err.Error())
	}

	logger.Info("sent OneSignal push notification to channel users")

	return nil
}

func DMNotification(db *gorm.DB, notifPayload models.NotificationProcessPayload, logger *utility.Logger) error {

	var (
		channelId = notifPayload.ChannelId
		orgId     = notifPayload.OrgId
		userId    = notifPayload.UserId
	)

	feed := notifPayload.Notification.Content.(models.FeedMessageRequest)

	typeCall := map[models.ChannelType]func() error{
		models.DMChannel: func() error {
			notifPayload.Notification.NotificationId = utility.GenerateUUID()
			err := centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", orgId, channelId), notifPayload.Notification)
			if err != nil {
				logger.Error(fmt.Sprintf("Error Publishing to participant id: %s, error: %v", channelId, err))
				return fmt.Errorf("failed to publish to participant")
			}

			logger.Info("published new_message notification to %d users in dm", 1)

			pushReq := models.PushRequest{
				ChannelName: feed.ChannelName,
				UserId:      channelId,
				Message:     feed.Content,
				TimeStamp:   feed.CreatedAt,
				AvatarUrl:   feed.AvatarURL,
				Title:       fmt.Sprintf("Notification from user %s", feed.ChannelName),
			}

			err = push_notifications.PushFCMToUser(pushReq, logger, db)
			if err != nil {
				logger.Error("Failed to send push notification to user %s: %v", channelId, err)
				return fmt.Errorf("failed to send push notification to user %s: %v", channelId, err)
			}

			// Send OneSignal notification to single DM user
			err = push_notifications.PushOneSignalToUser(pushReq, logger, db)
			if err != nil {
				logger.Error("Failed to send OneSignal notification to user %s: %v", channelId, err)
			}

			return nil
		},
		models.GroupDMChannel: func() error {

			var (
				userIDs []string
			)

			err := db.
				Model(&models.ChannelParticipant{}).
				Where("channel_id = ? AND user_id != ?", channelId, userId).
				Pluck("user_id", &userIDs).Error

			if err != nil {
				return fmt.Errorf("failed to fetch participants: %v", err)
			}

			if len(userIDs) == 0 {
				return nil
			}

			orgUserIds := make([]string, 0)

			for _, userId := range userIDs {
				orgUserIds = append(orgUserIds, fmt.Sprintf("%s/%s", orgId, userId))
			}

			notifPayload.Notification.NotificationId = utility.GenerateUUID()
			err = centrifuge.BatchBroadcastToChannel(logger, orgUserIds, notifPayload.Notification)
			if err != nil {
				logger.Error("Error Publishing to group_dm; channelid: %s, with orgid: %s error: %v", channelId, orgId, err.Error())
				return fmt.Errorf("failed to publish data")
			}

			logger.Info("published new_message notification to %d users in group_dm", len(userIDs))

			pushReq := models.PushRequest{
				UserIds:     userIDs,
				ChannelName: feed.ChannelName,
				Message:     feed.Content,
				TimeStamp:   feed.CreatedAt,
				AvatarUrl:   feed.AvatarURL,
				Title:       fmt.Sprintf("Notification from user %s", feed.ChannelName),
			}

			err = push_notifications.PushFCMToUsers(pushReq, logger, db)
			if err != nil {
				logger.Error("failed to send push notification to users %s: %v", userIDs, err)
			}

			// Send OneSignal notification to group DM users
			err = push_notifications.PushOneSignalToUsers(pushReq, logger, db)
			if err != nil {
				logger.Error("failed to send OneSignal notification to users %s: %v", userIDs, err)
			}

			logger.Info("sent fcm push notification to channel users")

			return nil
		},
	}

	return typeCall[notifPayload.ChannelType]()
}
