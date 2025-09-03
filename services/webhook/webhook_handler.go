package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
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
		resp     gin.H
		webhook  models.Webhook
		userChan models.UserChannels
		channel  models.Channels
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
		OrgansationID: channel.OrganisationID,
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

	err = centrifuge.PublishChannel(logger, webhook.ChannelId, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Publishing to channelid: %s, error: %v", webhook.ChannelId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
	}

	err = centrifuge.PublishChannel(logger, channel.OrganisationID, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Publishing to channelid: %s, error: %v", webhook.ChannelId, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
	}

	// increase unread count for channel users
	userChan.ChannelsID = webhook.ChannelId
	userChan.UserID = "00000000-0000-0000-0000-000000000000"
	userChan.OrgId = channel.OrganisationID

	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	// Add to the wait group for each goroutine that must complete first
	wg.Add(1)
	go func() {
		defer wg.Done()
		userChan.UpdateUnReadCount(db.Postgresql, mutex, logger)
	}()

	// Run this after the others finish
	go func() {
		wg.Wait()
		userChan.SendChannelUnReadUpdate(mutex, logger, models.NewThread, models.MentionMessage{})
	}()

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
		OrgansationID: channel.OrganisationID,
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

	err = centrifuge.PublishChannel(logger, req.ChannelID, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Publishing to channelid: %s, <: error: %v", req.ChannelID, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
	}

	err = centrifuge.PublishChannel(logger, channel.OrganisationID, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Publishing to channelid: %s, with orgid: %s <: error: %v", req.ChannelID, channel.OrganisationID, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to publish webhook data: " + err.Error())
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("Publishing to channelid: %s", req.ChannelID))

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
		OrgID:      req.OrgID,
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

	payload := []models.QueueFeed{feed}

	task := "telex_queue_processor.handle_new_message"

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Error marshaling payload for integration %s: %v", integration.ID, err.Error()))
		return fmt.Errorf("failed to marshal payload, error: %v", err)
	}

	err = rabbitmq.PushToRabbitQueue(logger, db, string(payloadBytes), routing_key, task)
	if err != nil {
		logger.Error(fmt.Sprintf("Error pushing to RabbitMQ for integration %s: %v", integration.ID, err.Error()))
		return fmt.Errorf("failed to push to RabbitMQ, error: %v", err)
	}

	logger.Info(fmt.Sprintf("Successfully pushed to RabbitMQ for integration %s", integration.Name))

	return nil
}
