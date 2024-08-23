package webhook

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateWebhook(req models.CreateWebhookRequest, db *gorm.DB) (models.Webhook, int, error) {

	webhook := models.Webhook{
		ID:        utility.GenerateUUID(),
		ChannelId: req.ChannelID,
		OwnerId:   req.UserID,
		Status:    "active",
	}

	slug := strings.Split(webhook.ID, "-")[4]
	webhookUrl := config.Config.App.WebhookApiUrl + fmt.Sprintf("/v1/webhooks/%s", slug)
	webhook.WebhookSlug = slug
	webhook.WebhookUrl = webhookUrl

	err := webhook.CreateWebhook(db)

	if err != nil {
		return webhook, http.StatusBadRequest, err
	}

	return webhook, http.StatusCreated, nil
}

func DeleteWebhook(req models.DeleteWebhookRequest, db *gorm.DB) (int, error) {

	webhook := models.Webhook{
		ChannelId: req.ChannelID,
		ID:        req.WebhookID,
		OwnerId:   req.UserID,
	}

	_, err := webhook.GetWebhookByID(db, webhook.ID, webhook.ChannelId)

	if err != nil {
		return http.StatusBadRequest, err
	}

	err = webhook.DeleteWebhook(db)

	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func UpdateWebhook(req models.UpdateWebhookRequest, db *gorm.DB) (models.Webhook, int, error) {

	var (
		resp models.Webhook
	)

	resp, err := resp.UpdateWebhook(db, req)

	if err != nil {
		return resp, http.StatusBadRequest, err
	}

	return resp, http.StatusOK, nil
}

func ChangeWebhookStatus(req models.ChangeWebhookStatusRequest, db *gorm.DB) (models.Webhook, int, error) {

	var (
		resp models.Webhook
	)

	resp, err := resp.UpdateWebhookStatus(db, req)

	if err != nil {
		return resp, http.StatusBadRequest, err
	}

	return resp, http.StatusOK, nil

}

func GetAllWebhook(db *gorm.DB, c *gin.Context, channelId string) ([]models.Webhook, postgresql.PaginationResponse, int, error) {

	var (
		resp     []models.Webhook
		webhooks models.Webhook
	)

	resp, pagResp, err := webhooks.GetAllChannelWebhook(db, c, channelId)

	if err != nil {
		return resp, pagResp, http.StatusBadRequest, err
	}

	return resp, pagResp, http.StatusOK, nil

}

func GetWebhookHistory(req models.GetWebhookHistoryRequest, c *gin.Context, db *gorm.DB) ([]models.HistoryWebhook, postgresql.PaginationResponse, int, error) {

	var (
		HistoryWebhook models.HistoryWebhook
	)

	resp, pagResp, code, err := HistoryWebhook.GetWebHookHistory(db, c, req)

	if err != nil {
		return resp, pagResp, code, err
	}

	return resp, pagResp, http.StatusOK, nil

}

func GetChannelWebhook(db *gorm.DB, c *gin.Context, channelId string) (models.Webhook, int, error) {

	var (
		resp     models.Webhook
		webhooks models.Webhook
	)

	resp, err := webhooks.GetChannelWebhook(db, channelId)

	if err != nil {
		return resp, http.StatusBadRequest, err
	}

	return resp, http.StatusOK, nil

}
