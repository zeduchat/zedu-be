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
	logger *utility.Logger,
) {
	userIds, err := threads.GetUsersInThread(userId)

	if err != nil {
		logger.Error("Error getting users in thread: %s, with orgid: %s error: %v", channelsId, orgId, err.Error())
	}

	feed := models.FeedMessageRequest{
		OrgId:     orgId,
		ChannelID: channelsId,
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
