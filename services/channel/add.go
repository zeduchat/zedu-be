package channel

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

func SaveChannelsMsg(req models.CreateMessageRequest, db *gorm.DB, typesenseDb *typesense.Client,
	logger *utility.Logger) (*models.Message, int, error) {

	var (
		profile models.Profile
		user    models.User
	)

	threadId, err := uuid.FromString(req.ThreadId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	message := models.Message{
		ID:         utility.GenerateUUID(),
		Content:    req.Content,
		ChannelsID: req.ChannelsId,
		UserID:     req.UserId,
		ThreadID:   threadId,
	}

	if err := message.CreateMessage(db, typesenseDb); err != nil {
		return &message, http.StatusBadRequest, errors.New("failed to save message, invalid thread id")
	}

	if err := thread.DetectAndAddMentions(message.ID, req.Content, db); err != nil {
		return &message, http.StatusBadRequest, err
	}

	err = profile.GetProfileByUserId(db, req.UserId)

	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user profile")
	}

	user, err = user.GetUserByID(db, req.UserId)

	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to get user")
	}

	feed := models.FeedMessageRequest{
		ChannelID: req.ChannelsId,
		UserName:  profile.UserName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AvatarURL: profile.AvatarURL,
		Type:      "message",
		Content:   req.Content,
		ThreadId:  req.ThreadId,
		Email:     user.Email,
		FullName:  profile.FullName,
	}

	err = centrifuge.BroadcastChannel(logger, req.ChannelsId, feed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", req.ChannelsId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	return &message, http.StatusCreated, nil
}

func EditChannelsMsg(req models.EditMessageRequest, db *gorm.DB, typesenseDb *typesense.Client,
	logger *utility.Logger) (*models.Message, int, error) {

	var message models.Message

	theMessage, err := message.GetMessageByID(db, req.MessageId)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid message ID")
	}

	theMessage.Content = req.Content
	theMessage.Edited = true
	newMsg, err := theMessage.UpdateMessage(db)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	if err := thread.DetectAndAddMentions(theMessage.ID, req.Content, db); err != nil {
		return nil, http.StatusBadRequest, err
	}

	return newMsg, http.StatusOK, nil
}

func AddChannelsMsg(req models.CreateMessageRequest, db *gorm.DB, typesenseDb *typesense.Client,
	logger *utility.Logger) (*models.Message, int, error) {

	var (
		routing_key = "new_message"
		oci         models.OrganisationChannelsIntegrations
	)

	res, err := oci.CheckHasFilterIntegrations(db, req.ChannelsId)

	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for integration filter status: %v", err.Error()))
		return &models.Message{}, http.StatusBadRequest, fmt.Errorf("failed fetching filter status, error: %v", err)
	}

	if !res {
		return SaveChannelsMsg(req, db, typesenseDb, logger)
	}

	returnUrl := fmt.Sprintf("%s/api/v1/channels/backend-queue", config.Config.App.Url)

	feed := models.FeedQueue{
		ChannelsId: req.ChannelsId,
		Content:    req.Content,
		ThreadId:   req.ThreadId,
		ReturnUrl:  returnUrl,
		Type:       "message",
		UserId:     req.UserId,
	}

	payload := map[string]interface{}{
		"args": []models.FeedQueue{feed},
		"task": "telex_queue_processor.handle_new_message",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshaling payload for integration: %v", err.Error()))
		return &models.Message{}, http.StatusBadRequest, fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db, string(payloadBytes), routing_key)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration: %v", err.Error()))
		return &models.Message{}, http.StatusBadRequest, fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	return &models.Message{}, http.StatusOK, nil
}

func SaveIncomingQueueMsg(req models.FeedQueue, db *gorm.DB, typesenseDb *typesense.Client,
	logger *utility.Logger) {

	var err error

	reqNew := models.CreateMessageRequest{
		Content:    req.Content,
		ChannelsId: req.ChannelsId,
		ThreadId:   req.ThreadId,
		UserId:     req.UserId,
	}

	if req.Type == "message" {

		_, _, err = SaveChannelsMsg(reqNew, db, typesenseDb, logger)
		
	} else {

		reqNew := models.CreateThreadMsgReq{
			Content:    req.Content,
			ChannelsID: req.ChannelsId,
			ThreadId:   req.ThreadId,
			UserId:     req.UserId,
		}
		_, err = thread.SaveThreadMessage(reqNew, db, typesenseDb, logger)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Error saving and broadcasting recieved message: %v", err.Error()))
		return
	}

	logger.Error("saving and broadcasting recieved message successfull !!!")
}
