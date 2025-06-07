package savedMessages

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func SaveMsgForLater(req models.CreateMessageRequest, db *storage.Database, logger *utility.Logger) (*models.MessageDocument, error) {
	var (
		profile       models.Profile
		channels      models.Channels
		user          models.User
		agent_message = false
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, errors.New("invalid thread ID")
	}

	chanExist := postgresql.CheckExists(db.Postgresql, &channels, "id = ?", req.ChannelsId)
	if !chanExist {
		return nil, errors.New("channel does not exist")
	}

	err = profile.GetProfileByUserId(db.Postgresql, req.UserId)
	if err != nil && !agent_message {
		logger.Error("failed to get user profile: %v", err)
		return nil, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db.Postgresql, req.UserId)
	if err != nil && !agent_message {
		logger.Error("failed to get user: %v", err)
		return nil, errors.New("failed to get user")
	}

	messageDoc := models.MessageDocument{
		ID:           utility.GenerateUUID(),
		Content:      req.Content,
		ChannelsID:   req.ChannelsId,
		UserID:       req.UserId,
		ThreadID:     threadId,
		AgentMessage: agent_message,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		AvatarURL:    profile.AvatarURL,
		Edited:       false,
		UserType:     "user",
		Username:     utility.ThisOrThat(profile.UserName, req.AgentName),
		FullName:     utility.ThisOrThat(profile.FullName, req.AgentName),
		Email:        user.Email,
		Media:        req.Media,
		Mentions:     req.Mentions,
	}

	_, createErr := messageDoc.CreateMessageForLater(db, logger)
	if createErr != nil {
		logger.Error("failed to save message: %v", createErr)
		return nil, errors.New("failed to save message, error: " + createErr.Error())
	}

	return &messageDoc, nil
}

func GetAllSavedMessages(db *storage.Database, logger *utility.Logger) ([]models.MessageDocument, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	var rawResponse interface{}

	err := elastic.SelectAll(db.Elastic, models.SavedMessageIndexName, query, &rawResponse)
	if err != nil {
		logger.Error("An error occurred while retrieving messages from elastic: %v", err)
		return nil, err
	}

	messageDoc, err := models.UnMarsahlMessageResponse(rawResponse)
	if err != nil {
		logger.Error("An error occurred while unmarshalling message response from elastic: %v", err)
		return nil, err
	}

	return messageDoc, nil
}

func DeleteSavedMessage(messageId string, db *storage.Database, logger *utility.Logger) error {
	var message models.MessageDocument

	err := message.DeleteSavedMessage(db, logger, messageId)
	if err != nil {
		return err
	}

	return nil
}
