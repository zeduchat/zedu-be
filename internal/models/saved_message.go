package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type SavedMessage struct {
	ID         string                 `gorm:"type:uuid;primary_key" json:"id"`
	Content    string                 `gorm:"column:content; type:text; not null" json:"content"`
	ChannelsID string                 `gorm:"type:uuid;not null;index" json:"channels_id"`
	UserID     string                 `gorm:"type:uuid;not null;index" json:"user_id"`
	CreatedAt  time.Time              `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt         `gorm:"index" json:"-"`
	ThreadID   uuid.UUID              `gorm:"type:uuid;null;index" json:"thread_id"`
	Media      []UploadedFileResponse `gorm:"type:jsonb;serializer:json" json:"media,omitempty"`
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

func (c *SavedMessage) DeleteSavedMessageMediaFiles(logger *utility.Logger, db *gorm.DB, mediaFiles []UploadedFileResponse) error {
	var (
		fileModel UploadedFileResponse
		firstErr  error
	)

	for _, mediaFile := range mediaFiles {
		count, countErr := fileModel.GetFileCountByLink(db, mediaFile.FileLink)
		if countErr != nil {
			logger.Error("Failed to get the number of files with the associated link:", countErr)
			if firstErr == nil {
				firstErr = countErr
			}
			continue
		}

		if count == 1 {
			hashedFileName := utility.ExtractHashedFileName(mediaFile.FileLink)

			err := DeleteUploadedFiles(logger, hashedFileName)
			if err != nil {
				logger.Error("Failed to delete uploaded file:", err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
		}

		deleteErr := mediaFile.DeleteFileByID(db, mediaFile.ID)
		if deleteErr != nil {
			logger.Error("Failed to delete DB file entry:", deleteErr)
			if firstErr == nil {
				firstErr = deleteErr
			}
			continue
		}
	}

	return firstErr
}

func (m *SavedMessage) GetSavedMessageByID(db *gorm.DB, messageID string) (*SavedMessage, error) {
	query := db.Where("id = ?", messageID)

	err := query.First(&m).Error
	if err != nil {
		return nil, err
	}

	return m, nil
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
