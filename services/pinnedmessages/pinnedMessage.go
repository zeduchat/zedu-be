package pinnedmessages

import (
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func PinThreadMessage(req models.PinMessageRequest, db *storage.Database, logger *utility.Logger) (*models.PinnedMessage, error) {
	var threads models.Threads

	threads.ID = req.ThreadId
	updateKey := map[string]interface{}{
		"is_pinned": true,
	}

	if resp, err := threads.UpdateThread(db.Postgresql, updateKey); err != nil {
		fmt.Printf("Response: %v", resp)
		return nil, err
	}

	messageToPin := models.PinnedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		PinnedAt:   time.Now().UTC(),
		ThreadID:   req.ThreadId,
	}

	createErr := messageToPin.CreatePinnedMessageRecord(db.Postgresql)
	if createErr != nil {
		logger.Error("failed to pin thread message: %v", createErr)
		return nil, errors.New("failed to pin thread message, error: " + createErr.Error())
	}

	return &messageToPin, nil
}

func PinReplyMessage(req models.PinMessageRequest, db *storage.Database, logger *utility.Logger) (*models.PinnedMessage, error) {
	var message models.Message

	updateKey := map[string]interface{}{
		"is_pinned": true,
	}

	message.ID = req.MessageID
	if _, err := message.UpdateMessage(db.Postgresql, updateKey); err != nil {
		return nil, err
	}

	messageToPin := models.PinnedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		PinnedAt:   time.Now().UTC(),
		ThreadID:   req.ThreadId,
		MessageID:  &req.MessageID,
	}

	createErr := messageToPin.CreatePinnedReplyMessageRecord(db.Postgresql)
	if createErr != nil {
		logger.Error("failed to pin reply-message: %v", createErr)
		return nil, errors.New("failed to pin reply-message, error: " + createErr.Error())
	}

	return &messageToPin, nil
}

func GetAllPinnedMessages(db *storage.Database, logger *utility.Logger, ids models.PinMessageRequestIds) ([]models.MessageDocument, error) {
	var pinnedMessage *models.PinnedMessage

	messageCollection, err := pinnedMessage.GetAllPinnedMessagesForChannel(db, ids)
	if err != nil {
		logger.Error("An error occurred while fetching pinned messages: %v", err)
		return nil, err
	}

	return messageCollection, nil
}

func UnPinThreadMessage(db *storage.Database, logger *utility.Logger, ids models.PinMessageRequestIds) error {
	var threads models.Threads
	var pinnedMessage models.PinnedMessage

	updateKey := map[string]interface{}{
		"is_pinned": false,
	}

	threads.ID = ids.ThreadId
	_, err := threads.UpdateThread(db.Postgresql, updateKey)
	if err != nil {
		return err
	}

	if err := pinnedMessage.DeletePinnedThreadMessageRecord(db.Postgresql, ids); err != nil {
		logger.Error("An error occurred while deleting pinned message record: %v", err)
		return err
	}

	return nil
}

func UnPinReplyMessage(db *storage.Database, logger *utility.Logger, ids models.PinMessageRequestIds) error {
	var message models.Message
	var pinnedMessage models.PinnedMessage

	updateKey := map[string]interface{}{
		"is_pinned": false,
	}

	message.ID = ids.MessageID
	_, err := message.UpdateMessage(db.Postgresql, updateKey)
	if err != nil {
		return err
	}

	if err := pinnedMessage.DeletePinnedReplyMessageRecord(db.Postgresql, ids); err != nil {
		logger.Error("An error occurred while deleting pinned message record: %v", err)
		return err
	}

	return nil
}
