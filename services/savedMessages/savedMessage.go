package savedMessages

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func SaveMsgForLater(req models.SaveMessageRequest, db *storage.Database, logger *utility.Logger) (*models.SavedMessage, error) {
	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	messageToSave := models.SavedMessage{
		ID:         utility.GenerateUUID(),
		Content:    req.Content,
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		Type:       req.Type,
		CreatedAt:  time.Now().UTC(),
		ThreadID:   threadId,
		Media:      req.Media,
	}

	createErr := messageToSave.CreateMessageRecord(db.Postgresql)
	if createErr != nil {
		logger.Error("failed to save message: %v", createErr)
		return nil, errors.New("failed to save message, error: " + createErr.Error())
	}

	return &messageToSave, nil
}

func GetAllSavedMessages(db *storage.Database, logger *utility.Logger, userId, orgId string) ([]models.SavedMessage, error) {
	var savedMessage *models.SavedMessage
	messageCollection, err := savedMessage.GetSavedMessages(db.Postgresql, userId, orgId)
	if err != nil {
		logger.Error("An error occurred while fetching messages from Postgres: %v", err)
		return nil, err
	}

	return messageCollection, nil
}

func DeleteSavedMessage(db *storage.Database, logger *utility.Logger, messageId, orgId, userId string) error {
	var savedMessage *models.SavedMessage

	message, err := savedMessage.GetSavedMessageByID(db.Postgresql, messageId, orgId, userId)
	if err != nil {
		logger.Error("An error occurred while fetching message from Postgres: %v", err)
		return err
	}

	mediaErr := savedMessage.DeleteSavedMessageMediaFiles(logger, db.Postgresql, message.Media)
	if mediaErr != nil {
		logger.Error("An error occurred while deleting media file: %v", mediaErr)
		return mediaErr
	}

	deleteErr := savedMessage.DeleteMessageByID(db.Postgresql, messageId, orgId)
	if deleteErr != nil {
		logger.Error("An error occurred while deleting saved message: %v", deleteErr)
		return deleteErr
	}

	return nil
}
