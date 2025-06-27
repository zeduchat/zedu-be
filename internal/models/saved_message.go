package models

import (
	"errors"
	"fmt"
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
	MessageID  *string        `gorm:"type:uuid;null;index" json:"message_id,omitempty"`
	ThreadID   uuid.UUID      `gorm:"type:uuid;null;index" json:"thread_id"`
	CreatedAt  time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type SavedMessagesResp struct {
	ID          string    `json:"id"`
	AvatarURL   string    `json:"avatar_url"`
	Username    string    `json:"username"`
	Content     string    `json:"content"`
	ChannelName string    `json:"channel_name"`
	Type        string    `json:"type"` // thread or message(thread-reply)
	SavedAt     time.Time `json:"saved_at"`
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

type SavedMessageIds struct {
	MessageID      string
	ThreadID       string
	OrgID          string
	UserID         string
	SavedMessageID string
}

func (m *SavedMessage) CreateMessageRecord(db *gorm.DB) error {
	var (
		org          Organisation
		dmChannels   DmChannels
		userChannels UserChannels
		savedMessage SavedMessage
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

	if chanExist {
		var channels Channels
		exists := postgresql.CheckExists(db, &channels, "id = ? AND org_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return errors.New("channel not found in organisation")
		}
	}

	if dmChanExist {
		var dmChannel DmChannels
		exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ? AND org_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return errors.New("direct message channel not found in organisation")
		}
	}

	threadExists := postgresql.CheckExists(db, &savedMessage, "org_id = ? AND user_id = ? AND thread_id = ?", m.OrgId, m.UserID, m.ThreadID)
	if threadExists {
		return errors.New("message to save already exists")
	}

	createErr := postgresql.CreateOneRecord(db, &m)
	if createErr != nil {
		return createErr
	}

	return nil
}

func (m *SavedMessage) CreateReplyMessageRecord(db *gorm.DB) error {
	var (
		org          Organisation
		dmChannels   DmChannels
		userChannels UserChannels
		savedMessage SavedMessage
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

	if chanExist {
		var channels Channels
		exists := postgresql.CheckExists(db, &channels, "id = ? AND org_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return errors.New("channel not found in organisation")
		}
	}

	if dmChanExist {
		var dmChannel DmChannels
		exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ? AND org_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return errors.New("direct message channel not found in organisation")
		}
	}

	msgExists := postgresql.CheckExists(db, &savedMessage, "org_id = ? AND user_id = ? AND thread_id = ? AND message_id = ?", m.OrgId, m.UserID, m.ThreadID, m.MessageID)
	if msgExists {
		return errors.New("message to save already exists")
	}

	createErr := postgresql.CreateOneRecord(db, &m)
	if createErr != nil {
		return createErr
	}

	return nil
}

func (m *SavedMessage) GetSavedMessageByID(db *gorm.DB, ids SavedMessageIds) error {
	var org Organisation

	exists := postgresql.CheckExists(db, &org, "id = ?", ids.OrgID)
	if !exists {
		return errors.New("organisation not found")
	}

	isMember, err := new(Organisation).CheckUserIsMemberOfOrg(ids.UserID, ids.OrgID, db)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a member of organisation")
	}

	err, _ = postgresql.SelectOneFromDb(db, &m, "id = ? AND org_id = ? AND user_id = ?", ids.SavedMessageID, ids.OrgID, ids.UserID)
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) DeleteMessageByID(db *gorm.DB) error {

	err := postgresql.HardDeleteRecordFromDb(db, m)
	if err != nil {
		return fmt.Errorf("failed to delete saved message: %w", err)
	}

	return nil
}

func (m *SavedMessage) GetSavedMessages(db *gorm.DB, ids SavedMessageIds) ([]SavedMessagesResp, error) {
	var (
		org          Organisation
		organisation *Organisation
		messages     []SavedMessage
		messagesResp []SavedMessagesResp
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", ids.OrgID)
	if !exists {
		return nil, errors.New("organisation not found")
	}

	isMember, err := organisation.CheckUserIsMemberOfOrg(ids.UserID, ids.OrgID, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of organisation")
	}

	findErr := db.Order("created_at DESC").Find(&messages).Where("org_id = ? AND user_id = ?", ids.OrgID, ids.UserID).Error
	if findErr != nil {
		return nil, findErr
	}

	for _, msg := range messages {
		var (
			t  ThreadDocument
			m  MessageDocument
			mr SavedMessagesResp
			ch Channels
		)

		if msg.MessageID != nil {
			err := m.GetMessageById(db, *msg.MessageID)
			if err != nil {
				continue
			}

			mr.ID = m.ID
			mr.AvatarURL = m.AvatarURL
			mr.Username = m.Username
			mr.Content = m.Content
			mr.SavedAt = msg.CreatedAt
			mr.Type = "message"

			exists := postgresql.CheckExists(db, &ch, "id = ?", m.ChannelsID)
			if !exists {
				mr.ChannelName = "Direct Message"
			}

			mr.ChannelName = ch.Name

			messagesResp = append(messagesResp, mr)

		} else {
			err := t.GetThreadById(db, msg.ThreadID.String())
			if err != nil {
				continue
			}

			mr.ID = t.ID
			mr.AvatarURL = t.AvatarURL
			mr.Username = t.Username
			mr.Content = t.Content
			mr.SavedAt = msg.CreatedAt
			mr.Type = "thread"

			exists := postgresql.CheckExists(db, &ch, "id = ?", t.ChannelsID)
			if !exists {
				mr.ChannelName = "Direct Message"
			} else {
				mr.ChannelName = ch.Name
			}

			messagesResp = append(messagesResp, mr)
		}
	}

	return messagesResp, nil
}

func (m *SavedMessage) DeleteSavedMessageByMessageID(db *gorm.DB, ids SavedMessageIds) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "message_id = ? AND thread_id = ? AND org_id = ?", ids.MessageID, ids.ThreadID, ids.OrgID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("message_id = ? AND thread_id = ? AND org_id = ? AND user_id = ?", ids.MessageID, ids.ThreadID, ids.OrgID, ids.UserID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) SavedThreadMsgExists(db *gorm.DB, ids SavedMessageIds) bool {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "thread_id = ? AND org_id = ?", ids.ThreadID, ids.OrgID)
	return idExists
}

func (m *SavedMessage) SavedReplyMsgExists(db *gorm.DB, ids SavedMessageIds) bool {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "message_id = ? AND thread_id = ? AND org_id = ?", ids.MessageID, ids.ThreadID, ids.OrgID)
	return idExists
}

func (m *SavedMessage) DeleteSavedThreadMsgByMessageID(db *gorm.DB, ids SavedMessageIds) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "thread_id = ? AND org_id = ?", ids.ThreadID, ids.OrgID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("thread_id = ? AND org_id = ? AND user_id = ?", ids.ThreadID, ids.OrgID, ids.UserID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}
