package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type SavedMessage struct {
	ID         string         `gorm:"type:uuid;primary_key" json:"id"`
	ChannelsID string         `gorm:"type:uuid;not null;index" json:"channels_id"`
	OrgId      string         `gorm:"type:uuid;not null;index" json:"org_id"`
	UserID     string         `gorm:"type:uuid;not null;index" json:"user_id"`
	Type       string         `gorm:"type:text;not null;index" json:"type,omitempty"`
	MessageID  uuid.UUID      `gorm:"type:uuid;null;index" json:"message_id,omitempty"`
	ThreadID   uuid.UUID      `gorm:"type:uuid;null;index" json:"thread_id"`
	CreatedAt  time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type SaveThreadRequest struct {
	ChannelsId string `json:"channels_id" validate:"required"`
	ThreadId   string `json:"thread_id" validate:"required"`
	Type       string `json:"type" validate:"required"`
	OrgId      string `json:"org_id"`
	UserId     string `json:"user_id"`
}

type SaveMessageRequest struct {
	ChannelsId string `json:"channels_id" validate:"required"`
	ThreadId   string `json:"thread_id" validate:"required"`
	MessageId  string `json:"message_id" validate:"required"`
	OrgId      string `json:"org_id"`
	UserId     string `json:"user_id"`
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

func (m *SavedMessage) DeleteSavedMessageByMessageID(db *gorm.DB, messageID, threadID, orgID string) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "message_id = ? AND thread_id = ? AND org_id = ?", messageID, threadID, orgID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("message_id = ? AND thread_id = ? AND org_id = ?", messageID, threadID, orgID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) DeleteSavedThreadMsgByMessageID(db *gorm.DB, threadID, orgID string) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "thread_id = ? AND org_id = ?", threadID, orgID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("thread_id = ? AND org_id = ?", threadID, orgID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}
