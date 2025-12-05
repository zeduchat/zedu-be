package savedMessages

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func SaveThreadForLater(req models.SaveThreadRequest, db *gorm.DB, logger *utility.Logger) (*models.SavedMessage, bool, error) {
	var (
		checkThread models.ThreadDocument
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, true, errors.New("invalid thread ID")
	}

	checkThread.ChannelsID = req.ChannelsId
	checkThread.ID = threadId.String()

	exists, _, err := checkThread.CheckExists()
	if err != nil {
		return nil, true, err
	}
	if !exists {
		return nil, true, errors.New("thread does not exist")
	}

	messageToSave := models.SavedMessage{
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		Type:       "thread",
		CreatedAt:  time.Now().UTC(),
		ThreadID:   threadId.String(),
	}

	saved, err := messageToSave.CreateSavedThreadRecord(db)
	if err != nil {
		logger.Error("failed to save thread message: %v", err)
		return nil, true, errors.New("failed to save thread message, error: " + err.Error())
	}

	if !saved {
		notification := models.Notification[models.UnSavedMessageEvent]
		notification.SectionType = models.ThreadSection
		notification.ModificationDetails = &models.ModificationDetails{
			ThreadId:  req.ThreadId,
			ChannelId: req.ChannelsId,
		}

		err = centrifuge.PublishChannel(logger, req.ChannelsId, notification)
		if err != nil {
			logger.Error("Error Publishing unsave message event with destination id: %s error: %v", req.ChannelsId, err.Error())
			return nil, false, errors.New("failed to publish data: " + err.Error())
		}

		return &messageToSave, false, nil
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
		return nil, true, errors.New("failed to publish data: " + err.Error())
	}

	return &messageToSave, true, nil

}

func SaveThreadReplyForLater(req models.SaveMessageRequest, db *gorm.DB, logger *utility.Logger) (*models.SavedMessage, bool, error) {
	var (
		checkMessage models.MessageDocument
	)

	messageID, err := uuid.FromString(req.MessageId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, true, errors.New("invalid thread ID")
	}

	checkMessage.ChannelsID = req.ChannelsId
	checkMessage.ID = messageID.String()

	exists, _, err := checkMessage.CheckExists()
	if err != nil {
		return nil, true, err
	}
	if !exists {
		return nil, true, errors.New("message does not exist")
	}

	messageToSave := models.SavedMessage{
		ChannelsID: req.ChannelsId,
		OrgId:      req.OrgId,
		UserID:     req.UserId,
		CreatedAt:  time.Now().UTC(),
		Type:       "reply",
		MessageID:  &req.MessageId,
		ThreadID:   req.ThreadId,
	}

	saved, createErr := messageToSave.CreateReplyMessageRecord(db)
	if createErr != nil {
		logger.Error("failed to save message: %v", createErr)
		return nil, true, errors.New("failed to save message, error: " + createErr.Error())
	}

	if !saved {
		notification := models.Notification[models.UnSavedMessageEvent]
		notification.SectionType = models.ThreadSection
		notification.ModificationDetails = &models.ModificationDetails{
			ThreadId:  req.ThreadId,
			ChannelId: req.ChannelsId,
			MessageId: req.MessageId,
		}

		err = centrifuge.PublishChannel(logger, req.ChannelsId, notification)
		if err != nil {
			logger.Error("Error Publishing unsaved message event to with destination id: %s error: %v", req.ChannelsId, err.Error())
			return nil, false, errors.New("failed to publish data: " + err.Error())
		}

		return &messageToSave, false, nil
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
		return nil, true, errors.New("failed to publish data: " + err.Error())
	}

	return &messageToSave, true, nil
}

func GetAllSavedMessages(db *gorm.DB, logger *utility.Logger, ids models.SavedMessageIds, pagination postgresql.Pagination) ([]models.SavedMessagesResp, postgresql.PaginationResponse, error) {
	var savedMessage *models.SavedMessage

	messageCollection, paginationResponse, err := savedMessage.GetSavedMessages(db, ids, pagination)
	if err != nil {
		logger.Error("An error occurred while fetching messages from Postgres: %v", err)
		return nil, postgresql.PaginationResponse{}, err
	}

	return messageCollection, paginationResponse, nil
}

func DeleteSavedMessage(db *gorm.DB, logger *utility.Logger, ids models.SavedMessageIds) error {
	var savedMessage models.SavedMessage

	err := savedMessage.GetSavedMessageByID(db, ids)
	if err != nil {
		logger.Error("An error occurred while fetching message from Postgres: %v", err)
		return err
	}

	ids.ChannelID = savedMessage.ChannelsID

	deleteErr := savedMessage.DeleteSavedMessageByID(db)
	if deleteErr != nil {
		logger.Error("An error occurred while deleting saved message: %v", deleteErr)
		return deleteErr
	}

	if savedMessage.Type == "thread" {
		notification := models.Notification[models.UnSavedMessageEvent]
		notification.SectionType = models.ThreadSection
		notification.ModificationDetails = &models.ModificationDetails{
			ThreadId:  ids.ThreadID,
			ChannelId: ids.ChannelID,
		}

		err = centrifuge.PublishChannel(logger, ids.ChannelID, notification)
		if err != nil {
			logger.Error("Error Publishing saved thread event to with destination id: %s error: %v", ids.ChannelID, err.Error())
			return errors.New("failed to publish data: " + err.Error())
		}
	}

	notification := models.Notification[models.UnSavedMessageEvent]
	notification.SectionType = models.ThreadSection
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  ids.ThreadID,
		ChannelId: ids.ChannelID,
		MessageId: ids.MessageID,
	}

	err = centrifuge.PublishChannel(logger, ids.ChannelID, notification)
	if err != nil {
		logger.Error("Error Publishing unsaved message event to with destination id: %s error: %v", ids.ChannelID, err.Error())
		return errors.New("failed to publish data: " + err.Error())
	}

	return nil
}
