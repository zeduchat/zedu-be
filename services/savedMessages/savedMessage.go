package savedMessages

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/utility"
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
	checkThread.ID = threadId.String()

	exists, _, err := checkThread.CheckExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("thread does not exist")
	}

	messageToSave := models.SavedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		Type:       "thread",
		CreatedAt:  time.Now().UTC(),
		ThreadID:   threadId,
	}

	err = messageToSave.CreateThreadMessageRecord(db)
	if err != nil {
		logger.Error("failed to save thread message: %v", err)
		return nil, errors.New("failed to save thread message, error: " + err.Error())
	}


	notification := models.Notification[models.SavedMessageEvent]
	notification.SectionType = models.ThreadSection
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsId, notification)
	if err != nil {
		logger.Error("Error Publishing saved message event to with destination id: %s error: %v", req.ChannelsId, err.Error())
		return nil, errors.New("failed to publish data: " + err.Error())
	}

	return &messageToSave, nil
}

func SaveReplyMessageForLater(req models.SaveMessageRequest, db *gorm.DB, logger *utility.Logger) (*models.SavedMessage, error) {
	var (
		checkMessage models.MessageDocument
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	checkMessage.ChannelsID = req.ChannelsId
	checkMessage.ID = threadId.String()

	exists, _, err := checkMessage.CheckExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("message does not exist")
	}

	messageToSave := models.SavedMessage{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		CreatedAt:  time.Now().UTC(),
		Type:       "reply",
		MessageID:  &req.MessageId,
		ThreadID:   threadId,
	}

	createErr := messageToSave.CreateReplyMessageRecord(db)
	if createErr != nil {
		logger.Error("failed to save message: %v", createErr)
		return nil, errors.New("failed to save message, error: " + createErr.Error())
	}

	notification := models.Notification[models.SavedMessageEvent]
	notification.SectionType = models.ThreadSection
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  req.ThreadId,
		ChannelId: req.ChannelsId,
	}

	err = centrifuge.PublishChannel(logger, req.ChannelsId, notification)
	if err != nil {
		logger.Error("Error Publishing saved message event to with destination id: %s error: %v", req.ChannelsId, err.Error())
		return nil, errors.New("failed to publish data: " + err.Error())
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

	deleteErr := savedMessage.DeleteSavedMessageByID(db)
	if deleteErr != nil {
		logger.Error("An error occurred while deleting saved message: %v", deleteErr)
		return deleteErr
	}

	return nil
}
