package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/services/rabbitmq"
	"github.com/hngprojects/telex_be/utility"
)

func PostWebhook(db *gorm.DB, logger *utility.Logger, req models.CreateWebhookHistoryRequest, typesenseDb *typesense.Client) (gin.H, int, error) {

	var (
		resp           gin.H
		webhook        models.Webhook
		HistoryWebhook models.HistoryWebhook
	)

	webhook, err := webhook.CheckExistBySlug(db, req.WebhookSlug)

	if err != nil {
		logger.Error("invalid webhook" + err.Error())
		return nil, http.StatusNotFound, errors.New("invalid webhook")
	}

	HistoryWebhook = models.HistoryWebhook{
		ID:          utility.GenerateUUID(),
		EventName:   req.EventName,
		WebhookID:   webhook.ID,
		WebhookSlug: req.WebhookSlug,
		ActionType:  req.ActionType,
		StatusCode:  "200",
		Retries:     int64(0),
	}
	err = HistoryWebhook.CreateWebhookHistory(db)
	if err != nil {
		logger.Error("failed to create webhook history" + err.Error())
	}

	thread := models.Threads{
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
	}
	err = thread.CreateThread(db, typesenseDb)
	if err != nil {
		logger.Error("failed to create webhook thread" + err.Error())
		return nil, http.StatusBadRequest, errors.New("failed to create new thread")
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

	err = centrifuge.BroadcastChannel(logger, webhook.ChannelId, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", webhook.ChannelId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	return resp, http.StatusOK, nil
}

func PostFeedWebhook(db *gorm.DB, logger *utility.Logger, req models.CreateWebhookHistoryRequest, typesenseDb *typesense.Client) (gin.H, int, error) {

	var (
		resp    gin.H
		channel models.Channels
	)

	_, err := channel.CheckChannelExists(db, req.ChannelID)

	if err != nil {
		logger.Error("error getting channel err: " + err.Error())
		return nil, http.StatusNotFound, errors.New("error getting channel, channel does not exist")
	}

	thread := models.Threads{
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
	}

	err = thread.CreateThread(db, typesenseDb)
	if err != nil {
		logger.Error("failed to create webhook thread" + err.Error())
		return nil, http.StatusBadRequest, errors.New("failed to create new thread")
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

	err = centrifuge.BroadcastChannel(logger, req.ChannelID, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", req.ChannelID, err.Error()))
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
		ReturnUrl:  fmt.Sprintf("%s/v1/backend-queue/return", base_url),
		Type:       "webhook",
		Content:    models.FeedWebHookRequest{
			ChannelID:  req.ChannelID,
			EventName:  req.EventName,
			UserName:   req.UserName,
			ActionType: req.ActionType,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			Status:     req.Status,
			AvatarURL:  req.AvatarURL,
			Type:       "webhook",
			Content:    req.Message,
		},
	}

	payload := map[string]interface{}{
		"args": []models.QueueFeed{feed},
		"task": "telex_queue_processor.handle_new_message",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(logger, fmt.Sprintf("Error marshaling payload for integration %s: %v", integration.ID, err.Error()))
		return fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db, string(payloadBytes), routing_key)
	if err != nil {
		logger.Error(logger, fmt.Sprintf("Error pushing to RabbitMQ for integration %s: %v", integration.ID, err.Error()))
		return fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	logger.Info(logger, fmt.Sprintf("Successfully pushed to RabbitMQ for integration %s", integration.Name))

	return nil
}
