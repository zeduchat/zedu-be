package savedMessages

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SaveThreadMessageForLater(req models.SaveThreadRequest, db *gorm.DB, logger *utility.Logger) (*models.SavedMessage, error) {
	var threads models.Threads
	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	messageToSave := models.SavedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		Type:       req.Type,
		CreatedAt:  time.Now().UTC(),
		ThreadID:   threadId,
	}

	createErr := messageToSave.CreateMessageRecord(db)
	if createErr != nil {
		logger.Error("failed to save thread message: %v", createErr)
		return nil, errors.New("failed to save thread message, error: " + createErr.Error())
	}

	threads.ID = req.ThreadId
	updateKey := map[string]any{
		"is_saved": true,
	}

	if _, err := threads.UpdateThread(db, updateKey); err != nil {
		return nil, err
	}

	return &messageToSave, nil
}

func SaveReplyMessageForLater(req models.SaveMessageRequest, db *gorm.DB, logger *utility.Logger) (*models.SavedMessage, error) {
	var message models.Message
	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	messageToSave := models.SavedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		CreatedAt:  time.Now().UTC(),
		MessageID:  &req.MessageId,
		ThreadID:   threadId,
	}

	createErr := messageToSave.CreateReplyMessageRecord(db)
	if createErr != nil {
		logger.Error("failed to save message: %v", createErr)
		return nil, errors.New("failed to save message, error: " + createErr.Error())
	}

	message.ID = req.MessageId
	updateKey := map[string]any{
		"is_saved": true,
	}

	if _, err := message.UpdateMessage(db, updateKey); err != nil {
		return nil, err
	}

	return &messageToSave, nil
}

func GetAllSavedMessages(db *gorm.DB, logger *utility.Logger, ids models.SavedMessageIds) ([]models.SavedMessagesResp, error) {
	var savedMessage *models.SavedMessage

	messageCollection, err := savedMessage.GetSavedMessages(db, ids)
	if err != nil {
		logger.Error("An error occurred while fetching messages from Postgres: %v", err)
		return nil, err
	}

	return messageCollection, nil
}

func DeleteSavedMessage(db *gorm.DB, logger *utility.Logger, ids models.SavedMessageIds) error {
	var savedMessage *models.SavedMessage

	_, err := savedMessage.GetSavedMessageByID(db, ids)
	if err != nil {
		logger.Error("An error occurred while fetching message from Postgres: %v", err)
		return err
	}

	deleteErr := savedMessage.DeleteMessageByID(db, ids)
	if deleteErr != nil {
		logger.Error("An error occurred while deleting saved message: %v", deleteErr)
		return deleteErr
	}

	return nil
}
