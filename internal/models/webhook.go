package models

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Webhook struct {
	ID             string           `gorm:"type:uuid;primary_key" json:"id"`
	EventName      string           `gorm:"column:event_name;type:text;null" json:"event_name"`
	WebhookName    string           `gorm:"column:webhook_name;type:text;null" json:"webhook_name"`
	Status         string           `gorm:"column:status; type:text;null" json:"status"`
	OwnerId        string           `gorm:"column:owner_id; type:uuid" json:"owner_id"`
	WebhookUrl     string           `gorm:"column:webhook_url; type:text; not null" json:"webhook_url"`
	WebhookSlug    string           `gorm:"column:webhook_slug; type:text;null" json:"webhook_slug"`
	ChannelId      string           `gorm:"column:channel_id; type:uuid" json:"channel_id"`
	CreatedAt      time.Time        `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt      time.Time        `gorm:"column: deleted_at; not null; autoDeleteTime" json:"deleted_at"`
	UpdatedAt      time.Time        `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	HistoryWebhook []HistoryWebhook `gorm:"foreignKey:WebhookID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"webhook_history"`
}

type HistoryWebhook struct {
	ID          string    `gorm:"column:id; type:uuid; primaryKey" json:"id"`
	CallbackID  string    `gorm:"column:callback_id; type:text;null" json:"callback_id"`
	EventName   string    `gorm:"column:event_name;type:text;null" json:"event_name"`
	WebhookSlug string    `gorm:"column:webhook_slug; type:text;null" json:"webhook_slug"`
	ActionType  string    `gorm:"column:action_type; type:text;null" json:"action_type"`
	StatusCode  string    `gorm:"column:status_code; type:text;null" json:"status_code"`
	WebhookID   string    `gorm:"type:uuid;not null" json:"-"`
	Retries     int64     `gorm:"type:integer;not null" json:"retries"`
	CreatedAt   time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"-"`
	UpdatedAt   time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"attempted"`
	DeletedAt   time.Time `gorm:"column: deleted_at; not null; autoDeleteTime" json:"-"`
}

type CreateWebhookRequest struct {
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
}

type UpdateWebhookRequest struct {
	ChannelID   string `json:"channel_id"`
	UserID      string `json:"user_id"`
	WebhookName string `json:"webhook_name" validate:"required"`
	WebhookID   string `json:"webhook_id"`
	EventName   string `json:"event_name"`
}

type DeleteWebhookRequest struct {
	ChannelID string `json:"channel_id"`
	WebhookID string `json:"webhook_id"`
	UserID    string `json:"user_id"`
}

type ChangeWebhookStatusRequest struct {
	ChannelID string `json:"channel_id"`
	WebhookID string `json:"webhook_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"webhook_status" validate:"required"`
}

type GetWebhookHistoryRequest struct {
	ChannelID string `json:"channel_id"`
	WebhookID string `json:"webhook_id"`
	UserID    string `json:"user_id"`
}

func (w *Webhook) CreateWebhook(db *gorm.DB) error {
	var (
		userChannel UserChannels
		webhook     Webhook
	)

	exist := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", w.ChannelId, w.OwnerId)
	if !exist {
		return errors.New("user not in channel")
	}

	exist = postgresql.CheckExists(db, &webhook, "channel_id = ?", w.ChannelId)

	if exist {
		return errors.New("webhook already exists")
	}

	err := postgresql.CreateOneRecord(db, w)
	if err != nil {
		return err
	}
	return nil
}

func (w *Webhook) DeleteWebhook(db *gorm.DB) error {
	var userChannel UserChannels

	exist := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", w.ChannelId, w.OwnerId)
	if !exist {
		return errors.New("user not in channel")
	}

	err := postgresql.DeleteRecordFromDb(db, &w)
	if err != nil {
		return errors.New("Failed to delete webhook: " + err.Error())
	}
	return nil
}

func (w *Webhook) UpdateWebhook(db *gorm.DB, req UpdateWebhookRequest) (Webhook, error) {
	var (
		userChannel UserChannels
		webhook     Webhook
	)

	exist := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", req.ChannelID, req.UserID)
	if !exist {
		return webhook, errors.New("user not in channel")
	}

	_, err := postgresql.UpdateFields(db, &webhook, req, "channel_id = ? AND id = ?", req.ChannelID, req.WebhookID)

	if err != nil {
		return webhook, err
	}

	webhook, err = webhook.GetWebhookByID(db, req.WebhookID, req.ChannelID)

	if err != nil {
		return webhook, err
	}

	return webhook, nil
}

func (w *Webhook) UpdateWebhookStatus(db *gorm.DB, req ChangeWebhookStatusRequest) (Webhook, error) {
	var (
		userChannel UserChannels
		webhook     Webhook
	)

	exist := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", req.ChannelID, w.OwnerId)
	if !exist {
		return webhook, errors.New("user not in channel")
	}

	_, err := postgresql.SaveAllFields(db, &w)
	if err != nil {
		return webhook, err
	}

	_, err = postgresql.UpdateFields(db, &webhook, req, "channel_id = ? AND id = ?", req.ChannelID, req.WebhookID)

	if err != nil {
		return webhook, err
	}

	webhook, err = webhook.GetWebhookByID(db, req.WebhookID, req.ChannelID)

	if err != nil {
		return webhook, err
	}

	return webhook, nil
}

func (r *Webhook) GetWebhookByID(db *gorm.DB, webhookID, channelId string) (Webhook, error) {

	var webhook Webhook

	_, err := postgresql.SelectOneFromDb(db, &webhook, "id = ? AND channel_id = ?", webhookID, channelId)
	if err != nil {
		return webhook, errors.New("error getting webhook by id: " + err.Error())
	}

	return webhook, nil
}

func (r *Webhook) GetAllChannelWebhook(db *gorm.DB, c *gin.Context, channelId string) ([]Webhook, postgresql.PaginationResponse, error) {
	var (
		webhooks []Webhook
	)

	pagination := postgresql.GetPagination(c)
	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"",
		"",
		pagination,
		&webhooks,
		"channel_id = ?",
		channelId,
	)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return webhooks, paginationResponse, errors.New("channel not found")
		}
		return webhooks, paginationResponse, err
	}

	return webhooks, paginationResponse, nil
}

func (wh *HistoryWebhook) GetWebHookHistory(db *gorm.DB, c *gin.Context, req GetWebhookHistoryRequest) ([]HistoryWebhook, postgresql.PaginationResponse, int, error) {
	var (
		HistoryWebhook []HistoryWebhook
		userChannel    UserChannels
	)

	exist := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", req.ChannelID, req.UserID)
	if !exist {
		return HistoryWebhook, postgresql.PaginationResponse{}, http.StatusInternalServerError, errors.New("user not in channel")
	}

	pagination := postgresql.GetPagination(c)
	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"",
		"",
		pagination,
		&HistoryWebhook,
		"webhook_id = ?",
		req.WebhookID,
	)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HistoryWebhook, paginationResponse, http.StatusBadRequest, errors.New("channel not found")
		}
		return HistoryWebhook, paginationResponse, http.StatusBadRequest, err
	}

	return HistoryWebhook, paginationResponse, http.StatusOK, nil
}

func (wh *HistoryWebhook) CreateWebhookHistory(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, wh)
	if err != nil {
		return err
	}

	return nil
}

func (r *Webhook) CheckExistBySlug(db *gorm.DB, webhookSlug string) (Webhook, error) {

	var webhook Webhook

	_, err := postgresql.SelectOneFromDb(db, &webhook, "webhook_slug = ?", webhookSlug)
	if err != nil {
		return webhook, errors.New("error getting webhook by id: " + err.Error())
	}

	exist := postgresql.CheckExists(db, &webhook, "webhook_slug = ?", webhookSlug)

	if !exist {
		return webhook, errors.New("webhook not found")
	}

	return webhook, nil
}

func (r *Webhook) GetChannelWebhook(db *gorm.DB, channelId string) (Webhook, error) {
	var (
		webhook Webhook
	)

	exist := postgresql.CheckExists(db, &webhook, "channel_id = ?", channelId)

	if !exist {
		return webhook, errors.New("webhook not found")
	}

	return webhook, nil
}
