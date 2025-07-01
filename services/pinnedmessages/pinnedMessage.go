package pinnedmessages

import (
	"errors"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func PinThreadMessage(req models.PinMessageRequest, db *storage.Database, logger *utility.Logger) (*models.PinnedMessage, int, error) {
	var (
		threads models.Threads
		user    models.User
	)

	messageToPin := models.PinnedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		UserID:     req.UserId,
		PinnedAt:   time.Now().UTC(),
		ThreadID:   req.ThreadId,
		Type:       "thread",
		Pinned:     true,
	}

	code, createErr := messageToPin.CreatePinnedThreadRecord(db.Postgresql)
	if createErr != nil {
		logger.Error("failed to pin thread message: %v", createErr)
		return nil, code, errors.New("failed to pin thread message, error: " + createErr.Error())
	}

	ud, _ := user.GetUserByID(db.Postgresql, req.UserId)
	pinnedDetails := models.PinnedDetails{
		Username: ud.Profile.UserName,
		Email:    ud.Email,
	}

	threads.ID = req.ThreadId
	updateKey := map[string]any{
		"is_pinned":      true,
		"pinned_details": pinnedDetails,
	}

	if _, err := threads.UpdateThread(db.Postgresql, updateKey); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	notification := models.Notification[models.PinnedMessageEvent]
	notification.SectionType = models.ThreadSection
	notification.PinnedDetails = &pinnedDetails
	notification.ModifcationDetails = &models.ModifcationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
	}

	err := centrifuge.PublishChannel(logger, req.ChannelsId, notification)
	if err != nil {
		logger.Error("Error Publishing pinned message event to with destination id: %s error: %v", req.ChannelsId, err.Error())
		return nil, http.StatusInternalServerError, errors.New("failed to publish data: " + err.Error())
	}

	return &messageToPin, code, nil
}

func PinReplyMessage(req models.PinMessageRequest, db *storage.Database, logger *utility.Logger) (*models.PinnedMessage, int, error) {
	var (
		message models.Message
		user    models.User
	)

	messageToPin := models.PinnedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		UserID:     req.UserId,
		PinnedAt:   time.Now().UTC(),
		ThreadID:   req.ThreadId,
		MessageID:  &req.MessageID,
		Type:       "reply",
		Pinned:     true,
	}

	code, createErr := messageToPin.CreatePinnedMessageRecord(db.Postgresql)
	if createErr != nil {
		logger.Error("failed to pin reply-message: %v", createErr)
		return nil, code, errors.New("failed to pin reply-message, error: " + createErr.Error())
	}

	ud, _ := user.GetUserByID(db.Postgresql, req.UserId)

	pinnedDetails := models.PinnedDetails{
		Username: ud.Profile.UserName,
		Email:    ud.Email,
	}

	updateKey := map[string]any{
		"is_pinned":      true,
		"pinned_details": pinnedDetails,
	}

	message.ID = req.MessageID
	if _, err := message.UpdateMessage(db.Postgresql, updateKey); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	notification := models.Notification[models.PinnedMessageEvent]
	notification.SectionType = models.ReplySection
	notification.PinnedDetails = &pinnedDetails
	notification.ModifcationDetails = &models.ModifcationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
		MessageId: req.MessageID,
	}
	err := centrifuge.PublishChannel(logger, req.ChannelsId, notification)
	if err != nil {
		logger.Error("Error Publishing pinned message event to with destination id: %s error: %v", req.ChannelsId, err.Error())
		return nil, http.StatusInternalServerError, errors.New("failed to publish data: " + err.Error())
	}

	return &messageToPin, code, nil
}

func GetAllPinnedMessages(db *storage.Database, logger *utility.Logger, ids models.IDS) ([]models.PinnedMessageResponse, error) {
	var pinnedMessage *models.PinnedMessage

	messageCollection, err := pinnedMessage.GetAllPinnedMessagesForChannel(db, ids)
	if err != nil {
		logger.Error("An error occurred while fetching pinned messages: %v", err)
		return nil, err
	}

	return messageCollection, nil
}

func UnPinThreadMessage(db *storage.Database, logger *utility.Logger, ids models.IDS) error {
	var threads models.Threads
	var pinnedMessage models.PinnedMessage

	exists := postgresql.CheckExists(db.Postgresql, &pinnedMessage, "type = ? AND channels_id = ? AND thread_id = ?", "thread", ids.ChannelID, ids.ThreadID)
	if !exists {
		return errors.New("thread not pinned")
	}

	pinnedMessage.ThreadID = ids.ThreadID
	pinnedMessage.ChannelsID = ids.ChannelID

	if err := pinnedMessage.DeletePinnedThreadMessageRecord(db.Postgresql); err != nil {
		logger.Error("An error occurred while deleting pinned message record: %v", err)
		return err
	}

	script := map[string]any{
		"script": map[string]any{
			"source": `
				ctx._source.is_pinned = false;
				ctx._source.pinned_details = [:];
			`,
		},
	}

	threads.ID = ids.ThreadID
	_, err := threads.UpdateThreadWithScript(db.Postgresql, script)
	if err != nil {
		return err
	}

	notification := models.Notification[models.UnPinnedMessageEvent]
	notification.SectionType = models.ThreadSection
	notification.ModifcationDetails = &models.ModifcationDetails{
		ThreadId:  pinnedMessage.ThreadID,
		ChannelId: pinnedMessage.ChannelsID,
	}

	err = centrifuge.PublishChannel(logger, pinnedMessage.ChannelsID, notification)
	if err != nil {
		logger.Error("Error Publishing unpinned message event to with destination id: %s error: %v", *&pinnedMessage.ChannelsID, err.Error())
		return errors.New("failed to publish data: " + err.Error())
	}

	return nil
}

func UnPinReplyMessage(db *storage.Database, logger *utility.Logger, ids models.IDS) error {
	var message models.Message
	var pinnedMessage models.PinnedMessage

	exists := postgresql.CheckExists(db.Postgresql, &pinnedMessage, "type = ? AND channels_id = ? AND message_id = ?", "reply", ids.ChannelID, ids.MessageID)
	if !exists {
		return errors.New("reply message not pinned")
	}

	pinnedMessage.MessageID = &ids.MessageID
	pinnedMessage.ChannelsID = ids.ChannelID

	if err := pinnedMessage.DeletePinnedReplyMessageRecord(db.Postgresql); err != nil {
		logger.Error("An error occurred while deleting pinned message record: %v", err)
		return err
	}

	script := map[string]any{
		"script": map[string]any{
			"source": `
				ctx._source.is_pinned = false;
				ctx._source.pinned_details = [:];
			`,
		},
	}

	message.ID = ids.MessageID
	_, err := message.UpdateMessageWithScript(db.Postgresql, script)
	if err != nil {
		return err
	}

	notification := models.Notification[models.UnPinnedMessageEvent]
	notification.SectionType = models.ReplySection
	notification.ModifcationDetails = &models.ModifcationDetails{
		ThreadId:  pinnedMessage.ThreadID,
		ChannelId: pinnedMessage.ChannelsID,
		MessageId: *pinnedMessage.MessageID,
	}

	err = centrifuge.PublishChannel(logger, pinnedMessage.ChannelsID, notification)
	if err != nil {
		logger.Error("Error Publishing unpinned message event to with destination id: %s error: %v", *&pinnedMessage.ChannelsID, err.Error())
		return errors.New("failed to publish data: " + err.Error())
	}

	return nil
}
