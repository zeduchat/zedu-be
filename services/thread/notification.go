package thread

import (
	"encoding/json"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/utility"
)

func TrackThreadNotification(
	userId string,
	channelsId string,
	orgId string,
	threads *models.ThreadDocument,
	feed models.FeedMessageRequest,
	logger *utility.Logger,
) {
	userIds, err := threads.GetUsersInThread(userId, feed.Mentions)

	if err != nil {
		logger.Error("Error getting users in thread: %s, with orgid: %s error: %v", channelsId, orgId, err.Error())
	}

	if storage.DB != nil && storage.DB.Postgresql != nil {
		go func(oID, tID string, uIDs []string) {
			if processErr := ProcessThreadUnseenForParticipants(storage.DB.Postgresql, logger, oID, tID, uIDs); processErr != nil {
				logger.Error("Error processing thread unseen for participants in thread: %s, with orgid: %s error: %v", tID, oID, processErr)
			}
		}(orgId, threads.ID, userIds)
	}

	if feed.OrgId == "" {
		feed.OrgId = orgId
	}
	if feed.ChannelID == "" {
		feed.ChannelID = channelsId
	}
	dataByte, _ := json.Marshal(feed)

	threadNotifRec := models.PushNotificationRecord{
		UserIds:     userIds,
		Sent:        false,
		Data:        string(dataByte),
		Type:        models.ThreadNotification,
		ChannelType: models.ThreadChannel,
		ChannelId:   channelsId,
		Section:     models.ThreadSection,
	}

	err = actions.AddPushNotificationToQueue(storage.DB.Redis, threadNotifRec)

	if err != nil {
		logger.Error("Error adding notification to channelid: %s, with orgid: %s error: %v", channelsId, orgId, err.Error())
	}

	logger.Info("added thread notification to queue for channel %s", channelsId)
}

const webhookSenderID = "00000000-0000-0000-0000-000000000000"

func BuildWebhookNotificationRecord(feed models.FeedMessageRequest) models.PushNotificationRecord {
	feed.UserId = webhookSenderID
	if feed.UserName == "" {
		feed.UserName = "Webhook"
	}
	if feed.Type == "" {
		feed.Type = "message"
	}
	dataByte, _ := json.Marshal(feed)

	return models.PushNotificationRecord{
		ChannelType: models.Channel,
		Data:        string(dataByte),
		Sent:        false,
		ChannelId:   feed.ChannelID,
		Section:     models.ThreadSection,
		Type:        models.NewMessage,
	}
}

func QueueWebhookNotification(feed models.FeedMessageRequest, logger *utility.Logger) {
	if storage.DB == nil || storage.DB.Redis == nil {
		return
	}

	err := actions.AddPushNotificationToQueue(storage.DB.Redis, BuildWebhookNotificationRecord(feed))
	if err != nil {
		logger.Error("Error adding webhook notification to channelid: %s, error: %v", feed.ChannelID, err.Error())
		return
	}

	logger.Info("added webhook notification to queue for channel %s", feed.ChannelID)
}
