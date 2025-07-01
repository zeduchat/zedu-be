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
	ThreadID    string    `json:"thread_id"`
	MessageID   *string   `json:"message_id,omitempty"`
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

func (m *SavedMessage) CreateMessageRecord(db *gorm.DB) (bool,error) {
	var (
		org          Organisation
		dmChannels   DmChannels
		userChannels UserChannels
		savedMessage SavedMessage
		threads      Threads
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", m.OrgId)
	if !exists {
		return true, errors.New("organisation not found")
	}

	isMember, err := new(Organisation).CheckUserIsMemberOfOrg(m.UserID, m.OrgId, db)
	if err != nil {
		return true, err
	}
	if !isMember {
		return true, errors.New("user is not a member of organisation")
	}

	chanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || chanExist) {
		return true, errors.New("user not in channel")
	}

	if chanExist {
		var channels Channels
		exists := postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return true, errors.New("channel not found in organisation")
		}
	}

	if dmChanExist {
		var dmChannel DmChannels
		exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return true, errors.New("direct message channel not found in organisation")
		}
	}

	threadExists := postgresql.CheckExists(db, &savedMessage, "org_id = ? AND user_id = ? AND thread_id = ?", m.OrgId, m.UserID, m.ThreadID)
	if threadExists {
		var threads Threads

		threads.ID = savedMessage.ThreadID.String()

		err := savedMessage.DeleteMessageByID(db)
		if err != nil {
			return false, err
		}

		updateKey := map[string]any{
			"is_saved": false,
		}

		if _, err := threads.UpdateThread(db, updateKey); err != nil {
			return false, err
		}

		return false,nil //unsaved
	}

	createErr := postgresql.CreateOneRecord(db, &m)
	if createErr != nil {
		return false, createErr
	}

	threads.ID = m.ThreadID.String()
	updateKey := map[string]any{
		"is_saved": true,
	}

	if _, err := threads.UpdateThread(db, updateKey); err != nil {
		return false, err
	}

	return true, nil //saved
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
		exists := postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return errors.New("channel not found in organisation")
		}
	}

	if dmChanExist {
		var dmChannel DmChannels
		exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return errors.New("direct message channel not found in organisation")
		}
	}

	msgExists := postgresql.CheckExists(db, &savedMessage, "org_id = ? AND user_id = ? AND thread_id = ? AND message_id = ?", m.OrgId, m.UserID, m.ThreadID, m.MessageID)
	if msgExists {
		var msg Message

		msg.ID = *savedMessage.MessageID

		err := savedMessage.DeleteMessageByID(db)
		if err != nil {
			return err
		}

		updateKey := map[string]any{
			"is_saved": false,
		}

		if _, err := msg.UpdateMessage(db, updateKey); err != nil {
			return err
		}

		return nil
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

	err, _ = postgresql.SelectOneFromDb(db, &m, "id = ?", ids.SavedMessageID)
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
		messagesResp = make([]SavedMessagesResp, 0)
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

	err = postgresql.SelectAllFromDb(db, "", &messages, "org_id = ? AND user_id = ?", ids.OrgID, ids.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve saved messages: %w", err)
	}

	resolveChannelName := func(db *gorm.DB, channelID string) string {
		var dmchan DmChannels
		if postgresql.CheckExists(db, &dmchan, "channel_id = ?", channelID) {
			return "Direct Message"
		}
		var ch Channels
		if postgresql.CheckExists(db, &ch, "id = ?", channelID) {
			return ch.Name
		}
		return ""
	}

	for _, msg := range messages {
		var (
			t  ThreadDocument
			m  MessageDocument
			mr SavedMessagesResp
		)

		if msg.MessageID != nil {
			if err := m.GetMessageById(db, *msg.MessageID); err != nil {
				continue
			}
			mr.ID = msg.ID
			mr.ThreadID = msg.ThreadID.String()
			mr.MessageID = msg.MessageID
			mr.AvatarURL = m.AvatarURL
			mr.Username = m.Username
			mr.Content = m.Content
			mr.SavedAt = msg.CreatedAt
			mr.Type = "message"
			mr.ChannelName = resolveChannelName(db, m.ChannelsID)
		} else {
			if err := t.GetThreadById(db, msg.ThreadID.String()); err != nil {
				continue
			}
			mr.ID = msg.ID
			mr.ThreadID = msg.ThreadID.String()
			mr.MessageID = nil
			mr.AvatarURL = t.AvatarURL
			mr.Username = t.Username
			mr.Content = t.Content
			mr.SavedAt = msg.CreatedAt
			mr.Type = "thread"
			mr.ChannelName = resolveChannelName(db, t.ChannelsID)
		}

		messagesResp = append(messagesResp, mr)
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
