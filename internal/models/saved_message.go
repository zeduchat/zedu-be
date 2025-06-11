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
	Content    string                 `gorm:"column:content; type:text; not null" json:"content",omitempty`
	ChannelsID string                 `gorm:"type:uuid;not null;index" json:"channels_id"`
	OrgId      string                 `gorm:"type:uuid;not null;index" json:"org_id"`
	UserID     string                 `gorm:"type:uuid;not null;index" json:"user_id"`
	Type       string                 `gorm:"type:text;not null;index" json:"type,omitempty"`
	CreatedAt  time.Time              `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt         `gorm:"index" json:"-"`
	ThreadID   uuid.UUID              `gorm:"type:uuid;null;index" json:"thread_id"`
	Media      []UploadedFileResponse `gorm:"type:jsonb;serializer:json" json:"media,omitempty"`
}

type SaveMessageRequest struct {
	ChannelsId string                 `json:"channels_id" validate:"required"`
	ThreadId   string                 `json:"thread_id" validate:"required"`
	Content    string                 `json:"content"`
	Type       string                 `json:"type"`
	OrgId      string                 `json:"org_id"`
	UserId     string                 `json:"user_id"`
	Media      []UploadedFileResponse `json:"media"`
}

func (m *SavedMessage) CreateMessageRecord(db *gorm.DB) error {
	var (
		org          Organisation
		dmChannels   DmChannels
		userChannels UserChannels
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", m.OrgId)
	if !exists {
		return errors.New("organisation not found")
	}

	isMember, err := new(Organisation).CheckUserIsMemberOfOrg(m.UserID, m.OrgId, db)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a member of organisation")
	}

	chanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || chanExist) {
		return errors.New("user not in channel")
	}

	createErr := postgresql.CreateOneRecord(db, &m)
	if createErr != nil {
		return createErr
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

func (m *SavedMessage) GetSavedMessageByID(db *gorm.DB, messageID, orgId, userId string) (*SavedMessage, error) {
	var org Organisation

	exists := postgresql.CheckExists(db, &org, "id = ?", orgId)
	if !exists {
		return nil, errors.New("organisation not found")
	}

	isMember, err := new(Organisation).CheckUserIsMemberOfOrg(userId, orgId, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of organisation")
	}

	query := db.Where("id = ? AND org_id = ?", messageID, orgId)

	findErr := query.First(&m).Error
	if findErr != nil {
		return nil, findErr
	}

	return m, nil
}

func (m *SavedMessage) DeleteMessageByID(db *gorm.DB, messageID, orgId string) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "id = ?", messageID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("id = ? AND org_id = ?", messageID, orgId)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) GetSavedMessages(db *gorm.DB, userId, orgId string) ([]SavedMessage, error) {
	var (
		org          Organisation
		organisation *Organisation
		messages     []SavedMessage
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", orgId)
	if !exists {
		return nil, errors.New("organisation not found")
	}

	isMember, err := organisation.CheckUserIsMemberOfOrg(userId, orgId, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of organisation")
	}

	findErr := db.Order("created_at DESC").Find(&messages).Where("org_id = ?", orgId).Error
	return messages, findErr
}
