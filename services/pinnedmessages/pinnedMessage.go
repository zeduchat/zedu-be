package pinnedmessages

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func PinMessage(req models.PinMessageRequest, db *storage.Database, logger *utility.Logger) (*models.PinnedMessage, error) {
	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	messageToPin := models.PinnedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		PinnedAt:   time.Now().UTC(),
		ThreadID:   threadId,
	}

	createErr := messageToPin.CreatePinnedMessageRecord(db.Postgresql)
	if createErr != nil {
		logger.Error("failed to save message: %v", createErr)
		return nil, errors.New("failed to save message, error: " + createErr.Error())
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
	var pinnedMessage *models.PinnedMessage

	_, getErr := pinnedMessage.GetPinnedMessageByID(db.Postgresql, pinnedID, orgID, channelID, userID)
	if getErr != nil {
		logger.Error("An error occurred while fetching pinned message record: %v", getErr)
		return getErr
	}

	deleteErr := pinnedMessage.DeletePinnedMessageRecord(db.Postgresql, pinnedID)
	if deleteErr != nil {
		logger.Error("An error occurred while deleting pinned message record: %v", deleteErr)
		return deleteErr
	}

	return nil
}
