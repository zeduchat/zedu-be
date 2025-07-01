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
	var (
		checkThread models.ThreadDocument
	)
	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	checkThread.ChannelsID = req.ChannelsId
	checkThread.UserId = req.UserId

	exists, _, err := checkThread.CheckExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("thread does not exist")
	}

	messageToSave := models.SavedMessage{
		ID:         req.ThreadId,
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		Type:       "thread",
		CreatedAt:  time.Now().UTC(),
		ThreadID:   threadId,
	}

	saved, createErr := messageToSave.CreateMessageRecord(db)
	if createErr != nil {
		logger.Error("failed to save thread message: %v", createErr)
		return nil, errors.New("failed to save thread message, error: " + createErr.Error())
	}

	if !saved {
		logger.Error("thread message already saved")
		return nil, nil
	}

	return &messageToSave, nil
}

func SaveReplyMessageForLater(req models.SaveMessageRequest, db *gorm.DB, logger *utility.Logger) (*models.SavedMessage, error) {
	var (
		message      models.Message
		checkMessage models.MessageDocument
	)
	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	checkMessage.ChannelsID = req.ChannelsId
	checkMessage.UserID = req.UserId

	exists, _, err := checkMessage.CheckExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("message does not exist")
	}

	messageToSave := models.SavedMessage{
		ID:         req.MessageId,
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		CreatedAt:  time.Now().UTC(),
		Type:       "message",
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
	savedMessage := &models.SavedMessage{}

	err := savedMessage.GetSavedMessageByID(db, ids)
	if err != nil {
		logger.Error("An error occurred while fetching message from Postgres: %v", err)
		return err
	}

	msgType := savedMessage.Type

	deleteErr := savedMessage.DeleteMessageByID(db)
	if deleteErr != nil {
		logger.Error("An error occurred while deleting saved message: %v", deleteErr)
		return deleteErr
	}

	if msgType == "thread" {
		var threads models.Threads

		threads.ID = savedMessage.ThreadID.String()

		updateKey := map[string]any{
			"is_saved": false,
		}

		if _, err := threads.UpdateThread(db, updateKey); err != nil {
			return err
		}
	} else {
		var message models.Message

		updateKey := map[string]any{
			"is_saved": false,
		}

		message.ID = *savedMessage.MessageID

		if _, err := message.UpdateMessage(db, updateKey); err != nil {
			return err
		}
	}

	return nil
}
