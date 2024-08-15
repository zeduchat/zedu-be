package webhook

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/utility"
)

func PostWebhook(db *gorm.DB, logger *utility.Logger, req models.CreateWebhookHistoryRequest) (gin.H, int, error) {

	var (
		resp           gin.H
		webhook        models.Webhook
		webhookHistory models.WebhookHistory
	)

	webhook, err := webhook.CheckExistBySlug(db, req.WebhookSlug)

	if err != nil {
		return nil, http.StatusNotFound, errors.New("invalid webhook")
	}

	webhookHistory = models.WebhookHistory{
		ID:          utility.GenerateUUID(),
		WebhookID:   webhook.ID,
		WebhookSlug: req.WebhookSlug,
		ActionType:  req.ActionType,
		StatusCode:  "200",
		Retries:     int64(0),
	}

	err = webhookHistory.CreateWebhookHistory(db)

	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to create webhook history")
	}

	// save to db

	feed := models.FeedWebHookRequest{
		ChannelID:  webhook.ChannelId,
		EventName:  webhook.EventName,
		UserName:   req.UserName,
		ActionType: req.ActionType,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Status:     "success",
	}

	err = centrifuge.BroadcastChannel(logger, webhook.ChannelId, feed)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", webhook.ChannelId, err.Error()))
		return nil, http.StatusInternalServerError, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	return resp, http.StatusOK, nil
}

func PostFeedWebhook(db *gorm.DB, logger *utility.Logger, req models.CreateWebhookHistoryRequest) (gin.H, int, error) {

	var (
		resp    gin.H
		webhook models.Webhook
	)

	feed := models.FeedWebHookRequest{
		ChannelID:  webhook.ChannelId,
		EventName:  webhook.EventName,
		UserName:   req.UserName,
		ActionType: req.ActionType,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	err := centrifuge.BroadcastChannel(logger, webhook.ChannelId, feed)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", webhook.ChannelId, err.Error()))
		return nil, http.StatusInternalServerError, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("Broadcasting to channelid: %s", webhook.ChannelId))

	return resp, http.StatusOK, nil
}
