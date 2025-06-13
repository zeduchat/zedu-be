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

func GetAllPinnedMessages(db *storage.Database, logger *utility.Logger, orgID, channelID, userID string) ([]models.PinnedMessage, error) {
	var pinnedMessage *models.PinnedMessage

	messageCollection, err := pinnedMessage.GetPinnedMessagesForChannel(db.Postgresql, orgID, channelID, userID)
	if err != nil {
		logger.Error("An error occurred while fetching pinned messages: %v", err)
		return nil, err
	}

	return messageCollection, nil
}

func UnPinMessage(db *storage.Database, logger *utility.Logger, pinnedID, orgID, channelID, userID string) error {
	var threads models.Threads
	var message models.Message
	var pinnedMessage models.PinnedMessage

	getErr := pinnedMessage.GetPinnedMessageByID(db.Postgresql, pinnedID, orgID, channelID, userID)
	if getErr != nil {
		logger.Error("An error occurred while fetching pinned message record: %v", getErr)
		return getErr
	}

	updateKey := map[string]interface{}{
		"is_pinned": false,
	}

	if pinnedMessage.MessageID != nil {
		message.ID = *pinnedMessage.MessageID
		_, err := message.UpdateMessage(db.Postgresql, updateKey)
		if err != nil {
			return err
		}
	} else {
		threads.ID = pinnedMessage.ThreadID
		_, err := threads.UpdateThread(db.Postgresql, updateKey)
		if err != nil {
			return err
		}
	}

	deleteErr := pinnedMessage.DeletePinnedMessageRecord(db.Postgresql, pinnedID)
	if deleteErr != nil {
		logger.Error("An error occurred while deleting pinned message record: %v", deleteErr)
		return deleteErr
	}

	return nil
}
