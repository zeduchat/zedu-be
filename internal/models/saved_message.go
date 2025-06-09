package models

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
	"time"
)

type SavedMessage struct {
	ID         string         `gorm:"type:uuid;primary_key" json:"id"`
	Content    string         `gorm:"column:content; type:text; not null" json:"content"`
	ChannelsID string         `gorm:"type:uuid;not null;index" json:"channels_id"`
	UserID     string         `gorm:"type:uuid;not null;index" json:"user_id"`
	CreatedAt  time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	ThreadID   uuid.UUID      `gorm:"type:uuid;null;index" json:"thread_id"`
}

type SaveMessageRequest struct {
	Content    string                 `json:"content" validate:"required"`
	OrgId      string                 `json:"org_id" validate:"required"`
	ChannelsId string                 `json:"channels_id" validate:"required"`
	ThreadId   string                 `json:"thread_id" validate:"required"`
	UserId     string                 `json:"user_id"`
	Media      []UploadedFileResponse `json:"media"`
}

func (m *SavedMessage) CreateMessageRecord(db *gorm.DB) error {
	var (
		dmChannels   DmChannels
		userChannels UserChannels
	)

	chanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || chanExist) {
		return errors.New("user not in channel")
	}

	err := postgresql.CreateOneRecord(db, &m)
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) DeleteMessageByID(db *gorm.DB, messageID string) error {
	var savedMessage SavedMessage
	idExists := postgresql.CheckExists(db, &savedMessage, "id = ?", messageID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("id = ?", messageID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) GetSavedMessages(db *gorm.DB) ([]SavedMessage, error) {
	var messages []SavedMessage

	result := db.Order("created_at DESC").Find(&messages)
	return messages, result.Error
}
