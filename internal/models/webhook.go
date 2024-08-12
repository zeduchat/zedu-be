package models

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type Webhook struct {
	ID             string           `gorm:"type:uuid;primary_key" json:"room_id"`
	EventName      string           `gorm:"column:event_name;unique type:text; not null" json:"event_name"`
	Status         string           `gorm:"column:status; type:text; not null" json:"status"`
	WebhookUrl     string           `gorm:"column:webhook_url; type:text; not null" json:"webhook_url"`
	OwnerId        string           `gorm:"column:owner_id; type:uuid" json:"owner_id"`
	ChannelId      string           `gorm:"column:channel_id; type:uuid" json:"channel_id"`
	CreatedAt      time.Time        `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt      time.Time        `gorm:"column: deleted_at; not null; autoDeleteTime" json:"deleted_at"`
	UpdatedAt      time.Time        `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	WebhookHistory []WebhookHistory `gorm:"foreignKey:WebhookID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"webhook_history"`
}

type WebhookHistory struct {
	ID          int       `gorm:"column:id; type:serial; primaryKey" json:"id"`
	CallbackID  string    `gorm:"column:callback_id; type:text; not null" json:"callback_id"`
	WebhookSlug string    `gorm:"column:webhook_slug; type:text; not null" json:"webhook_id"`
	ActionType  string    `gorm:"column:action_type; type:text; not null" json:"action_type"`
	StatusCode  string    `gorm:"column:status_code; type:text; not null" json:"status_code"`
	WebhookID   string    `gorm:"type:uuid;not null" json:"-"`
	Retries     int64     `gorm:"type:integer;not null" json:"user_id"`
	CreatedAt   time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"attempted"`
	UpdatedAt   time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt   time.Time `gorm:"column: deleted_at; not null; autoDeleteTime" json:"deleted_at"`
}

type CreateWebhookRequest struct {
	ChannelID string `json:"channel_id" validate:"required"`
	UserID    string `json:"user_id"`
}

func (w *Webhook) CreateWebhook(db *gorm.DB) error {
	var userRoom UserRoom

	exist := postgresql.CheckExists(db, &userRoom, "room_id = ? AND user_id = ?", w.ChannelId, w.OwnerId)
	if !exist {
		return errors.New("user not in channel")
	}

	err := postgresql.CreateOneRecord(db, w)
	if err != nil {
		return err
	}
	return nil
}

func (r *Webhook) GetWebhookByID(db *gorm.DB, roomID string) ([]UserRoom, error) {
	var users []UserRoom

	err := postgresql.SelectUsersFromDb(
		db.Where("room_id = ?", roomID),
		"",
		&users,
		"room_id = ?",
		roomID,
	)

	if err != nil {
		return users, errors.New("could not get users in room")
	}

	return users, nil
}
