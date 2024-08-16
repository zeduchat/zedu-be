package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
)

type Message struct {
	ID         string     `gorm:"type:uuid;primary_key" json:"id"`
	Content    string     `gorm:"column:content; type:text; not null" json:"content"`
	ChannelsID string     `gorm:"type:uuid;not null;index" json:"channels_id"`
	UserID     string     `gorm:"type:uuid;not null;index" json:"user_id"`
	Username   string     `gorm:"column:username; type:varchar(100)" json:"username"`
	CreatedAt  time.Time  `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	ThreadID   uuid.UUID  `gorm:"type:uuid;null;index" json:"thread_id"`
	Mentions   []Mentions `gorm:"foreignKey:MessageID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"mentions"`
}

type CreateMessageRequest struct {
	Content    string `json:"content" validate:"required"`
	UserId     string `json:"user_id"`
	ChannelsId string `json:"channels_id"`
	ThreadId   string `json:"thread_id,omitempty"`
}

func (m *Message) CreateMessage(db *gorm.DB, typesenseDb *typesense.Client) error {
	var userChannels UserChannels

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	if !exist {
		return errors.New("user not in channel")
	}

	m.Username = userChannels.Username

	err := postgresql.CreateOneRecord(db, m)
	if err != nil {
		return err
	}

	threadId := m.ThreadID.String()
	messageDocument := ChannelDocument{
		ID:           m.ID,
		Type:         "message",
		ChannelsID:   m.ChannelsID,
		ThreadID:     threadId,
		UserID:       m.UserID,
		Username:     m.Username,
		Content:      m.Content,
		CreatedAt:    m.CreatedAt.Unix(),
		EventName:    "",
		ActionType:   "",
		Status:       "",
		ThreadStatus: "",
		MessageCount: 0,
	}

	err = tydb.InsertDocument(typesenseDb, m.ChannelsID, messageDocument)
	if err != nil {
		return errors.New("could not create message document in Typesense")
	}

	return nil
}

func (m *Message) GetMessagesByChannelsID(db *gorm.DB, userId, channelID string) ([]Message, error) {
	var messages []Message
	var userChannels UserChannels

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userId)
	if !exist {
		return messages, errors.New("user not in channel")
	}

	err := postgresql.SelectAllFromDb(db, "", &messages, "channels_id = ?", channelID)
	if err != nil {
		return messages, err
	}
	return messages, nil
}

func (m *Message) GetMessageByID(db *gorm.DB, messageID string) (Message, error) {
	var message Message

	err, nerr := postgresql.SelectOneFromDb(db, &message, "message_id = ?", messageID)
	if err != nil {
		return message, nerr
	}
	return message, nil
}
