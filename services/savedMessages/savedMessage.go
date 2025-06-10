package savedMessages

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func SaveMsgForLater(req models.SaveMessageRequest, db *storage.Database, logger *utility.Logger) (*models.SavedMessage, error) {
	var (
		channels models.Channels
		org      models.Organisation
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		logger.Error("invalid thread ID")
		return nil, errors.New("invalid thread ID")
	}

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	if !chanExist {
		logger.Error("channel does not exist")
		return nil, errors.New("channel does not exist")
	}

	orgExist := postgresql.CheckExists(db.Postgresql, &org, "id = ?", req.OrgId)
	if !orgExist {
		logger.Error("organisation does not exist")
		return nil, errors.New("organisation does not exist")
	}
	
	isMember, err := org.CheckUserIsMemberOfOrg(req.UserId, req.OrgId, db.Postgresql)
	if err != nil {
		logger.Error("an error occurred, %v", err)
		return nil, err
	}
	if !isMember {
		logger.Error("user is not a member of this organisation")
		return nil, errors.New("user is not a member of this organisation")
	}

	messageToSave := models.SavedMessage{
		ID:         utility.GenerateUUID(),
		Content:    req.Content,
		ChannelsID: req.ChannelsId,
		UserID:     req.UserId,
		CreatedAt:  time.Now().UTC(),
		ThreadID:   threadId,
	}

	createErr := messageToSave.CreateMessageRecord(db.Postgresql)
	if createErr != nil {
		logger.Error("failed to save message: %v", createErr)
		return nil, errors.New("failed to save message, error: " + createErr.Error())
	}

	return &messageToSave, nil
}

func GetAllSavedMessages(db *storage.Database, logger *utility.Logger) ([]models.SavedMessage, error) {
	var savedMessage *models.SavedMessage
	messageCollection, err := savedMessage.GetSavedMessages(db.Postgresql)
	if err != nil {
		logger.Error("An error occurred while fetching messages from Postgres: %v", err)
		return nil, err
	}

	return messageCollection, nil
}

func DeleteSavedMessage(messageId string, db *storage.Database, logger *utility.Logger) error {
	var savedMessage *models.SavedMessage

	err := savedMessage.DeleteMessageByID(db.Postgresql, messageId)
	if err != nil {
		logger.Error("An error occurred while deleting message with id %v from Postgres", err)
		return err
	}

	return nil
}
