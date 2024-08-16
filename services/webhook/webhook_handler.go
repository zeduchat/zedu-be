package webhook

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
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
		ID:         utility.GenerateUUID(),
		ChannelsID: webhook.ChannelId,
		EventName:  req.EventName,
		Username:   req.UserName,
		ActionType: req.ActionType,
		Status:     "success",
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
		Status:     "success",
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
	)

	thread := models.Threads{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelID,
		EventName:  req.EventName,
		Username:   req.UserName,
		ActionType: req.ActionType,
		Status:     "success",
	}

	err := thread.CreateThread(db, typesenseDb)
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
		Status:     "success",
	}

	err = centrifuge.BroadcastChannel(logger, req.ChannelID, feed)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error Broadcasting to channelid: %s, error: %v", req.ChannelID, err.Error()))
		return nil, http.StatusBadRequest, errors.New("failed to broadcast webhook data: " + err.Error())
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("Broadcasting to channelid: %s", req.ChannelID))

	return resp, http.StatusOK, nil
}
