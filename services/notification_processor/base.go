package notification_processor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	notificationpref "github.com/hngprojects/telex_be/services/notification_pref"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/utility"
)

func stripHTMLTags(content string) string {
	htmlEntities := map[string]string{
		"&nbsp;":   " ",
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&#39;":    "'",
		"&apos;":   "'",
		"&ndash;":  "-",
		"&mdash;":  "—",
		"&hellip;": "...",
	}

	for entity, replacement := range htmlEntities {
		content = strings.ReplaceAll(content, entity, replacement)
	}

	content = strings.ReplaceAll(content, "</p>", "\n")
	content = strings.ReplaceAll(content, "<br>", "\n")
	content = strings.ReplaceAll(content, "<br/>", "\n")
	content = strings.ReplaceAll(content, "<br />", "\n")

	re := regexp.MustCompile(`<[^>]*>?`)
	content = re.ReplaceAllString(content, "")

	content = strings.TrimSpace(content)

	maxLength := 100
	if len(content) > maxLength {
		content = content[:maxLength] + "..."
	}

	return content
}

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
		UserIds:      notification.UserIds,
	}

	typeCall := map[models.ChannelType]func(db *gorm.DB, notification models.NotificationProcessPayload, logger *utility.Logger) error{
		models.Channel:        ChannelNotification,
		models.DMChannel:      DMNotification,
		models.GroupDMChannel: DMNotification,
		models.ThreadChannel:  ThreadNotification,
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
		Pluck("user_id", &userIDs).Error

	if err != nil {
		return fmt.Errorf("failed to query entry of userids")
	}

	filteredUserIDs, err := notificationpref.FilterUsersByPreferences(db, userIDs, channelId, orgId, notificationpref.NotificationTypeAllMessages)
	if err != nil {
		logger.Error("failed to filter users by preferences: %v", err.Error())
		return err
	}

	orgUserIds := make([]string, 0)

	for _, userId := range filteredUserIDs {
		orgUserIds = append(orgUserIds, fmt.Sprintf("%s/%s", orgId, userId))
	}

	if len(orgUserIds) == 0 {
		logger.Info("Channel Notification aborted, empty users after preference filtering")
		return nil
	}

	notifPayload.Notification.NotificationId = utility.GenerateUUID()

	err = centrifuge.BatchBroadcastToChannel(logger, orgUserIds, notifPayload.Notification)
	if err != nil {
		logger.Error("Error Publishing to channelid: %s, with orgid: %s error: %v", channelId, orgId, err.Error())
		return fmt.Errorf("failed to publish thread data")
	}

	logger.Info("published new_message notification to %d users", len(filteredUserIDs))

	// Push fcm notification to channel users

	feed := notifPayload.Notification.Content.(models.FeedMessageRequest)

	pushReq := models.PushRequest{
		ChannelId:   channelId,
		OrgId:       orgId,
		ChannelName: feed.ChannelName,
		UserIds:     filteredUserIDs,
		Message:     stripHTMLTags(feed.Content),
		UserId:      userId,
		Username:    utility.ThisOrThat(feed.UserName, strings.Split(feed.Email, "@")[0]),
		Title:       fmt.Sprintf("Notification from user %s", feed.ChannelName),
	}

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
			shouldSend, checkErr := notificationpref.ShouldSendNotification(db, channelId, channelId, orgId, notificationpref.NotificationTypeDirectMessage)
			if checkErr != nil {
				logger.Error("Failed to check notification preferences for user %s: %v", channelId, checkErr)
				return checkErr
			}

			if !shouldSend {
				logger.Info("DM notification skipped for user %s due to preferences", channelId)
				return nil
			}

			notifPayload.Notification.NotificationId = utility.GenerateUUID()
			err := centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", orgId, channelId), notifPayload.Notification)
			if err != nil {
				logger.Error(fmt.Sprintf("Error Publishing to participant id: %s, error: %v", channelId, err))
				return fmt.Errorf("failed to publish to participant")
			}

			logger.Info("published new_message notification to %d users in dm", 1)

			pushReq := models.PushRequest{
				ChannelId:   channelId,
				OrgId:       orgId,
				ChannelName: feed.ChannelName,
				UserId:      channelId,
				Message:     stripHTMLTags(feed.Content),
				TimeStamp:   feed.CreatedAt,
				AvatarUrl:   feed.AvatarURL,
				Title:       feed.ChannelName,
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

			filteredUserIDs, err := notificationpref.FilterUsersByPreferences(db, userIDs, channelId, orgId, notificationpref.NotificationTypeDirectMessage)
			if err != nil {
				logger.Error("failed to filter users by preferences: %v", err.Error())
				return err
			}

			if len(filteredUserIDs) == 0 {
				logger.Info("Group DM notification aborted, no users after preference filtering")
				return nil
			}

			orgUserIds := make([]string, 0)

			for _, userId := range filteredUserIDs {
				orgUserIds = append(orgUserIds, fmt.Sprintf("%s/%s", orgId, userId))
			}

			notifPayload.Notification.NotificationId = utility.GenerateUUID()
			err = centrifuge.BatchBroadcastToChannel(logger, orgUserIds, notifPayload.Notification)
			if err != nil {
				logger.Error("Error Publishing to group_dm; channelid: %s, with orgid: %s error: %v", channelId, orgId, err.Error())
				return fmt.Errorf("failed to publish data")
			}

			logger.Info("published new_message notification to %d users in group_dm", len(filteredUserIDs))

			pushReq := models.PushRequest{
				ChannelId:   channelId,
				OrgId:       orgId,
				UserIds:     filteredUserIDs,
				ChannelName: feed.ChannelName,
				Message:     stripHTMLTags(feed.Content),
				TimeStamp:   feed.CreatedAt,
				AvatarUrl:   feed.AvatarURL,
				Title:       fmt.Sprintf("Notification from user %s", feed.ChannelName),
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

func ThreadNotification(db *gorm.DB, notifPayload models.NotificationProcessPayload, logger *utility.Logger) error {

	var (
		orgId   = notifPayload.OrgId
		userIds = notifPayload.UserIds
	)

	for _, userId := range userIds {
		threadCtx := &gin.Context{
			Request: &http.Request{
				URL: &url.URL{
					RawQuery: "page=1&limit=10",
				},
			},
		}

		var t models.Threads
		t.UserId = userId
		t.OrganisationID = orgId

		res, _, err := t.GetUserThreadsByOrganization(threadCtx, db, logger)
		if err != nil {
			logger.Error("Error fetching user threads for user %s: %v", userId, err)
			continue
		}

		notifPayload.Notification.Content = res
		notifPayload.Notification.NotificationId = utility.GenerateUUID()

		err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", orgId, userId), notifPayload.Notification)
		if err != nil {
			logger.Error("Error publishing thread notification for user %s: %v", userId, err)
		}

		logger.Info("published thread notification to user %s", userId)
	}

	logger.Info("published thread notification to %d users", len(userIds))

	return nil
}
