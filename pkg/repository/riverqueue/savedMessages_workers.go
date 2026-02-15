package riverqueueBg

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	"github.com/hngprojects/telex_be/utility"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
)

type SavedMessagesRemainderWorker struct {
	logger *utility.Logger
	db     *gorm.DB
	river.WorkerDefaults[models.SavedMessagesRemainderJobArgs]
}

func (w *SavedMessagesRemainderWorker) Work(ctx context.Context, job *river.Job[models.SavedMessagesRemainderJobArgs]) error {
	var (
		c       = job.Args
		content models.SetRemainderRequest
	)

	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Recovered in SavedMessagesRemainderWorker Work from panic: %v", r)
			debug.PrintStack()
		}
	}()

	w.logger.Info("Processing SavedMessagesRemainderJob for UserID: %s, OrgID: %s, ChannelsID: %s", job.Args.UserID, job.Args.OrgId, job.Args.ChannelsID)

	logger := w.logger

	notification := models.Notification[models.SavedMessageRemainder]

	if c.MessageID != nil {
		content.MessageId = c.MessageID
		notification.SectionType = models.ThreadSection
	}
	content.UserId = c.UserID
	content.OrgId = c.OrgId
	content.Type = c.Type
	content.ThreadId = c.ThreadID

	notification.NotificationId = utility.GenerateUUID()
	notification.Content = content
	notification.SectionType = models.ReplySection

	err := centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", c.OrgId, content.UserId), notification)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Publishing to channelid: %s, with userid: %s error: %v", c.ChannelsID, content.UserId, err.Error()))
		return err
	}

	logger.Info("Successfully Published Remainder Notification to channelid: %s, with userid: %s", c.ChannelsID, content.UserId)

	resolveChannelName := func(db *gorm.DB, channelID string) string {
		var dmchan models.DmChannels
		if postgresql.CheckExists(db, &dmchan, "channel_id = ?", channelID) {
			return "Direct Message"
		}
		var ch models.Channels
		if postgresql.CheckExists(db, &ch, "id = ?", channelID) {
			return ch.Name
		}

		return "unknown"
	}

	//send push notification
	pushRequest := models.PushRequest{
		ChannelId:   c.ChannelsID,
		OrgId:       c.OrgId,
		UserId:      content.UserId,
		ChannelName: resolveChannelName(w.db, c.ChannelsID),
		Message:     "You have a saved message reminder.",
		Title:       "Saved Messages Reminder",
		Payload:     notification,
	}

	err = push_notifications.PushOneSignalToUser(pushRequest, logger, w.db)
	if err != nil {
		logger.Error("Failed to send push notification for saved message remainder to user %s: %v", content.UserId, err)
		return err
	}

	logger.Info("Sent push notification for saved message remainder to user %s", content.UserId)

	return nil
}
