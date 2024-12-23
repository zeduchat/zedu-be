package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/utility"
)

func PostWebhook(db *storage.Database, logger *utility.Logger, req models.CreateWebhookHistoryRequest) (gin.H, int, error) {

	var (
		resp    gin.H
		webhook models.Webhook
		channel models.Channels
	)

	webhook, err := webhook.CheckExistBySlug(db.Postgresql, req.WebhookSlug)

	if err != nil {
		logger.Error("invalid webhook" + err.Error())
		return nil, http.StatusNotFound, errors.New("invalid webhook")
	}

	ch, err := channel.CheckChannelExists(db.Postgresql, webhook.ChannelId)

	if !ch || err != nil {
		logger.Error("invalid webhook" + err.Error())
		return nil, http.StatusInternalServerError, errors.New("channel does not exist")
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		ChannelsID:    webhook.ChannelId,
		EventName:     req.EventName,
		Username:      req.UserName,
		ActionType:    req.ActionType,
		Status:        req.Status,
		AvatarURL:     req.AvatarURL,
		Type:          "thread",
		Content:       req.Message,
		CurrentStatus: "pending",
		CreatedAt:     time.Now().UTC(),
		ChannelName:   channel.Name,
		Messages:      []models.MessageDocument{},
		MessageCount:  0,
	}

	err = threadDoc.CreateThread(db, logger)

	if err != nil {
		logger.Error("failed to create webhook thread" + err.Error())
		return nil, http.StatusInternalServerError, err
	}

	feed := models.FeedWebHookRequest{
		ChannelID:  webhook.ChannelId,
		EventName:  req.EventName,
		UserName:   req.UserName,
		ActionType: req.ActionType,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Status:     req.Status,
		AvatarURL:  req.AvatarURL,
		Type:       "thread",
		Content:    req.Message,
	}

	err = centrifuge.BroadcastChannel(logger, channel.OrganisationID, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", webhook.ChannelId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	return resp, http.StatusOK, nil
}

func PostFeedWebhook(db *storage.Database, logger *utility.Logger, req models.CreateWebhookHistoryRequest) (gin.H, int, error) {

	var (
		resp    gin.H
		channel models.Channels
	)

	_, err := channel.CheckChannelExists(db.Postgresql, req.ChannelID)

	if err != nil {
		logger.Error("error getting channel err: " + err.Error())
		return nil, http.StatusNotFound, errors.New("error getting channel, channel does not exist")
	}

	threadDoc := models.ThreadDocument{
		ID:            utility.GenerateUUID(),
		ChannelsID:    req.ChannelID,
		EventName:     req.EventName,
		Username:      req.UserName,
		ActionType:    req.ActionType,
		Status:        req.Status,
		AvatarURL:     req.AvatarURL,
		Type:          "thread",
		Content:       req.Message,
		CurrentStatus: "pending",
		CreatedAt:     time.Now().UTC(),
		ChannelName:   channel.Name,
		Messages:      []models.MessageDocument{},
		MessageCount:  0,
	}

	err = threadDoc.CreateThread(db, logger)

	if err != nil {
		logger.Error("failed to create webhook thread" + err.Error())
		return nil, http.StatusInternalServerError, err
	}

	feed := models.FeedWebHookRequest{
		ChannelID:  req.ChannelID,
		EventName:  req.EventName,
		UserName:   req.UserName,
		ActionType: req.ActionType,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Status:     req.Status,
		AvatarURL:  req.AvatarURL,
		Type:       "thread",
		Content:    req.Message,
	}

	err = centrifuge.BroadcastChannel(logger, channel.OrganisationID, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Broadcasting to channelid: %s, with orgid: %s <: error: %v", req.ChannelID, channel.OrganisationID,  err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("Broadcasting to channelid: %s", req.ChannelID))

	return resp, http.StatusOK, nil
}

func PostWebhookQueue(db *gorm.DB, logger *utility.Logger, req models.CreateWebhookHistoryRequest) error {
	var (
		integration models.Integrations
		routing_key = "new_message"
		base_url    = config.Config.App.Url
	)

	feed := models.QueueFeed{
		ChannelsId: req.ChannelID,
		OrgID: req.OrgID,
		ReturnUrl:  fmt.Sprintf("%s/v1/webhooks/backend-queue/return", base_url),
		Content: models.FeedWebHookRequest{
			ChannelID: req.ChannelID,
			EventName: req.EventName,
			UserName:  req.UserName,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Status:    req.Status,
			AvatarURL: req.AvatarURL,
			Type:      "webhook",
			Content:   req.Message,
		},
	}

	payload := map[string]interface{}{
		"args": []models.QueueFeed{feed},
		"task": "telex_queue_processor.handle_new_message",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshaling payload for integration %s: %v", integration.ID, err.Error()))
		return fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db, string(payloadBytes), routing_key)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration %s: %v", integration.ID, err.Error()))
		return fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	logger.Info(fmt.Sprintf("Successfully pushed to RabbitMQ for integration %s", integration.Name))

	return nil
}
