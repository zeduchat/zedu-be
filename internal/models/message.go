package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

var MessageIndexName = "messages"

type Message struct {
	ID         string         `gorm:"type:uuid;primary_key" json:"id"`
	Content    string         `gorm:"column:content; type:text; not null" json:"content"`
	ChannelsID string         `gorm:"type:uuid;not null;index" json:"channels_id"`
	UserID     string         `gorm:"type:uuid;not null;index" json:"user_id"`
	Username   string         `gorm:"column:username; type:varchar(100)" json:"username"`
	CreatedAt  time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	ThreadID   uuid.UUID      `gorm:"type:uuid;null;index" json:"thread_id"`
	Mentions   []Mentions     `gorm:"foreignKey:MessageID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"mentions"`
	AvatarURL  string         `json:"avatar_url,omitempty"`
	Edited     bool           `gorm:"type:bool" json:"edited,omitempty"`
}

type MessageDocument struct {
	ID         string         `json:"id"`
	Content    string         `json:"message"`
	ChannelsID string         `json:"channels_id"`
	UserID     string         `json:"user_id"`
	Username   string         `json:"user_name"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-"`
	ThreadID   uuid.UUID      `json:"thread_id"`
	AvatarURL  string         `json:"avatar_url"`
	Edited     bool           `json:"edited"`
	FullName   string         `json:"full_name"`
	Email      string         `json:"email"`
}

var MessageMapping = map[string]interface{}{
	"properties": map[string]interface{}{
		"id":          map[string]string{"type": "keyword"},
		"channels_id": map[string]string{"type": "keyword"},
		"user_id":     map[string]string{"type": "keyword"},
		"username":    map[string]string{"type": "keyword"},
		"thread_id":   map[string]string{"type": "keyword"},
		"avatar_url":  map[string]string{"type": "text"},
		"edited":      map[string]string{"type": "boolean"},
		"content":     map[string]string{"type": "text"},
		"created_at": map[string]string{
			"type":   "date",
			"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
		},
		"updated_at": map[string]string{
			"type":   "date",
			"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
		},
		"deleted_at": map[string]string{
			"type":   "date",
			"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
		},
	},
}

type CreateMessageRequest struct {
	Content    string `json:"content" validate:"required"`
	UserId     string `json:"user_id"`
	ChannelsId string `json:"channels_id"`
	ThreadId   string `json:"thread_id"`
}

type EditMessageRequest struct {
	Content    string `json:"content" validate:"required"`
	UserId     string `json:"user_id"`
	ChannelsId string `json:"channels_id"`
	ThreadId   string `json:"thread_id"`
	MessageId  string `json:"message_id" validate:"required"`
}

func (m *MessageDocument) CreateMessage(db *storage.Database, logger *utility.Logger) error {
	var (
		userChannels UserChannels
		profile      Profile
		thread       Threads
	)

	exist := postgresql.CheckExists(db.Postgresql, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	if !exist {
		return errors.New("user not in channel")
	}

	m.Username = userChannels.Username

	err := elastic.AddDocument(db.Elastic, MessageIndexName, m.ID, interface{}(&m), logger)
	if err != nil {
		return err
	}

	_, err = thread.GetThreadById(db.Postgresql, m.ThreadID.String())

	if len(thread.Messages) < 5 {

		script := `if (ctx._source.messages == null) {
			ctx._source.messages = [];
		}
		boolean found = false;
		for (int i = 0; i < ctx._source.messages.size(); i++) {
			if (ctx._source.messages[i] != null && ctx._source.messages[i].user_id == params.message.user_id) {
				ctx._source.messages[i] = params.message;
				found = true;
				break;
			}
		}
		if (!found) {
			ctx._source.messages.add(params.message);
		}
		ctx._source.message_count++;
		ctx._source.last_reply = params.message.created_at;`

		req := map[string]interface{}{
			"script": map[string]interface{}{
				"source": script,
				"params": map[string]interface{}{
					"message": &m,
				},
			},
		}

		err = elastic.UpdateDocWithScript(db.Elastic, ThreadIndexName, m.ThreadID.String(), req)
		if err != nil {
			logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
			return err
		}

	} else {

		script := `ctx._source.message_count++;
		ctx._source.last_reply = params.created_at;`

		req := map[string]interface{}{
			"script": map[string]interface{}{
				"source": script,
				"params": map[string]interface{}{
					"created_at": &m.CreatedAt,
				},
			},
		}

		err = elastic.UpdateDocWithScript(db.Elastic, ThreadIndexName, m.ThreadID.String(), req)
		if err != nil {
			logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
			return err
		}

	}

	err = profile.GetProfileByUserId(db.Postgresql, m.UserID)
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) UpdateMessage(db *gorm.DB) (*Message, error) {
	result, err := postgresql.SaveAllFields(db, &m)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update message")
	}

	return m, nil
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

	err, nerr := postgresql.SelectOneFromDb(db, &message, "id = ?", messageID)
	if err != nil {
		return message, nerr
	}
	return message, nil
}

func (t *Message) GetAllMessagesByThreadID(c *gin.Context, db *gorm.DB, userId, ThreadID string) ([]Message, *elastic.PaginationResponse, error) {
	var (
		messages []Message
	)

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit

	from := (page - 1) * limit

	// Build the query
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"thread_id.keyword": ThreadID,
			},
		},
		"from": from,
		"size": limit,
		"sort": []map[string]interface{}{
			{
				"created_at": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	var messageData interface{}

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, MessageIndexName, query, &messageData, c)

	if err != nil {
		return nil, pagR, errors.New(fmt.Sprintf("failed to fetch message records, error: %v", err))
	}

	messages, err = UnMarsahlMessageResponse(messageData)
	if err != nil {
		return nil, pagR, err
	}

	return messages, pagR, nil
}

func UnMarsahlMessageResponse(messageData interface{}) (messages []Message, err error) {

	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source Message `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	rawJSON, _ := json.MarshalIndent(messageData.(map[string]interface{}), "", "  ")

	if errr := json.Unmarshal(rawJSON, &searchResult); errr != nil {
		err = errors.New(fmt.Sprintf("failed to unmarshal result, error: %v", errr))
		return
	}

	messages = make([]Message, len(searchResult.Hits.Hits))

	for i, hit := range searchResult.Hits.Hits {
		messages[i] = hit.Source
	}

	return
}
