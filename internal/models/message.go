package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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
	ID            string         `gorm:"type:uuid;primary_key" json:"id"`
	Content       string         `gorm:"column:content; type:text; not null" json:"content"`
	ChannelsID    string         `gorm:"type:uuid;not null;index" json:"channels_id"`
	UserID        string         `gorm:"type:uuid;not null;index" json:"user_id"`
	Username      string         `gorm:"column:username; type:varchar(100)" json:"username"`
	CreatedAt     time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	ThreadID      uuid.UUID      `gorm:"type:uuid;null;index" json:"thread_id"`
	Mentions      []Mentions     `gorm:"foreignKey:MessageID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"mentions,omitempty"`
	AvatarURL     string         `json:"avatar_url,omitempty"`
	Edited        bool           `gorm:"type:bool" json:"edited,omitempty"`
	QuotedMessage *QuotedMessage `json:"quoted_message,omitempty"`
}

type MessageDocument struct {
	ID             string                 `json:"id",omitempty`
	Content        string                 `json:"message"`
	OrganisationID string                 `json:"org_id"`
	ChannelsID     string                 `json:"channels_id"`
	UserID         string                 `json:"user_id"`
	Username       string                 `json:"username"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	DeletedAt      gorm.DeletedAt         `json:"-"`
	AgentMessage   bool                   `json:"-"`
	UserType       string                 `json:"user_type"`
	ThreadID       uuid.UUID              `json:"thread_id"`
	AvatarURL      string                 `json:"avatar_url"`
	Edited         bool                   `json:"edited"`
	FullName       string                 `json:"full_name"`
	Email          string                 `json:"email"`
	Media          []UploadedFileResponse `json:"media,omitempty"`
	Mentions       []Mention              `json:"mentions,omitempty"`
	QuotedMessage  *QuotedMessage         `json:"quoted_message,omitempty"`
}

type QuotedMessage struct {
	ThreadID  string    `json:"thread_id"`
	Content   string    `json:"message"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

var MessageMapping = map[string]interface{}{
	"properties": map[string]interface{}{
		"id":          map[string]string{"type": "keyword"},
		"channels_id": map[string]string{"type": "keyword"},
		"user_id":     map[string]string{"type": "keyword"},
		"org_id":      map[string]string{"type": "keyword"},
		"username":    map[string]string{"type": "keyword"},
		"user_type":   map[string]string{"type": "keyword"},
		"thread_id":   map[string]string{"type": "keyword"},
		"avatar_url":  map[string]string{"type": "text"},
		"edited":      map[string]string{"type": "boolean"},
		"message":     map[string]string{"type": "text"},
		"full_name":   map[string]string{"type": "text"},
		"email":       map[string]string{"type": "text"},
		"created_at": map[string]string{
			"type":   "date",
			"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
		},
		"media": map[string]interface{}{
			"type":       "nested",
			"properties": MediaMapping,
		},
		"mention": map[string]interface{}{
			"type":       "nested",
			"properties": MentionMapping,
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
	Content    string                 `json:"content" validate:"required"`
	UserId     string                 `json:"user_id"`
	ChannelsId string                 `json:"channels_id"`
	ThreadId   string                 `json:"thread_id" validate:"required"`
	OrgId      string                 `json:"org_id"`
	AgentName  string                 `json:"agent_name"`
	Media      []UploadedFileResponse `json:"media"`
	Mentions   []Mention              `json:"mentions"`
}

type EditMessageRequest struct {
	Content    string `json:"content" validate:"required"`
	UserId     string `json:"user_id"`
	ChannelsId string `json:"channels_id"`
	ThreadId   string `json:"thread_id" validate:"required"`
	MessageId  string `json:"message_id" validate:"required"`
	OrgId      string `json:"org_id"`
}

func (m *MessageDocument) CreateMessage(db *storage.Database, logger *utility.Logger) (map[string]interface{}, error) {
	var (
		dmChannels   DmChannels
		userChannels UserChannels
		thread       ThreadDocument
	)

	updateResp := map[string]interface{}{}
	previewSect := false

	chanExist := postgresql.CheckExists(db.Postgresql, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db.Postgresql, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || chanExist) && !m.AgentMessage {
		return updateResp, errors.New("user not in channel")
	}

	err := elastic.AddDocument(db.Elastic, MessageIndexName, m.ID, interface{}(&m), logger)
	if err != nil {
		return updateResp, err
	}

	err = thread.GetThreadById(db.Postgresql, m.ThreadID.String())

	if err != nil {
		return updateResp, err
	}

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
			return updateResp, err
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
			return updateResp, err
		}

	}

	err = thread.GetThreadById(db.Postgresql, m.ThreadID.String())
	if err != nil {
		return updateResp, err
	}

	for _, con := range thread.Messages {
		if con.ID == m.ID {
			previewSect = true
		}
	}

	updateResp["thread_count"] = thread.MessageCount
	updateResp["preview_section"] = previewSect

	return updateResp, nil
}

func (m *Message) UpdateMessage(db *gorm.DB, req map[string]interface{}) (*Message, error) {

	err := elastic.UpdateDocument(storage.DB.Elastic, MessageIndexName, m.ID, req)

	if err != nil {
		return nil, fmt.Errorf("message not found")
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

func (t *MessageDocument) GetMessageById(db *gorm.DB, messageID string) error {

	var (
		messageData interface{}
	)

	err := elastic.SelectByID(storage.DB.Elastic, MessageIndexName, messageID, &messageData)

	if err != nil {
		return fmt.Errorf("failed to fetch message records, error: %v", err)
	}

	rawJSON, _ := json.MarshalIndent(messageData.(map[string]interface{}), "", "  ")

	if err := json.Unmarshal(rawJSON, &t); err != nil {
		return fmt.Errorf("failed to decode search response: %v", err)

	}

	return nil
}

func (m *MessageDocument) DeleteMessage(db *gorm.DB, logger *utility.Logger) (map[string]interface{}, error) {

	var (
		thread ThreadDocument
	)

	updateResp := map[string]interface{}{}
	previewSect := false

	err := elastic.DeleteDocument(storage.DB.Elastic, MessageIndexName, m.ID)

	if err != nil {
		return updateResp, fmt.Errorf("failed to delete message, err: %v", err)
	}

	err = thread.GetThreadById(db, m.ThreadID.String())

	if err != nil {
		return updateResp, err
	}

	for _, con := range thread.Messages {
		if con.ID == m.ID {
			previewSect = true
		}
	}

	if previewSect {
		script := `if (ctx._source.messages == null) {
			ctx._source.messages = [];
		}
		boolean found = false;
		for (int i = 0; i < ctx._source.messages.size(); i++) {
			if (ctx._source.messages[i] != null && ctx._source.messages[i].id == params.message_id) {
				ctx._source.messages.remove(i);
				found = true;
				break;
			}
		}

		if (found) {
			ctx._source.message_count--;
		}`

		req := map[string]interface{}{
			"script": map[string]interface{}{
				"source": script,
				"params": map[string]interface{}{
					"message_id": m.ID,
				},
			},
		}

		err = elastic.UpdateDocWithScript(storage.DB.Elastic, ThreadIndexName, m.ThreadID.String(), req)
		if err != nil {
			logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
			return updateResp, err
		}

	} else {

		script := `if (ctx._source.message_count > 0) {
			ctx._source.message_count--;
		}`
		req := map[string]interface{}{
			"script": map[string]interface{}{
				"source": script,
			},
		}

		err = elastic.UpdateDocWithScript(storage.DB.Elastic, ThreadIndexName, m.ThreadID.String(), req)
		if err != nil {
			logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
			return updateResp, err
		}
	}

	err = thread.GetThreadById(db, m.ThreadID.String())

	if err != nil {
		return updateResp, err
	}

	updateResp["thread_count"] = thread.MessageCount
	updateResp["preview_section"] = previewSect

	return updateResp, nil
}

func (c *Message) DeleteMessageMediaFiles(logger *utility.Logger, db *gorm.DB, mediaFiles []UploadedFileResponse) (*Message, error) {
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

	return c, firstErr
}

func (t *Message) GetAllMessagesByThreadID(c *gin.Context, db *gorm.DB, userId, ThreadID string) ([]MessageDocument, *elastic.PaginationResponse, error) {
	var (
		messages []MessageDocument
		thread   ThreadDocument
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
		return nil, pagR, fmt.Errorf("failed to fetch message records, error: %v", err)
	}

	err = thread.GetThreadById(db, ThreadID)
	if err != nil {
		return nil, pagR, err
	}

	messages, err = UnMarsahlMessageResponse(messageData)
	if err != nil {
		return nil, pagR, err
	}

	return messages, pagR, nil
}

func UnMarsahlMessageResponse(messageData interface{}) (messages []MessageDocument, err error) {

	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source MessageDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	rawJSON, _ := json.MarshalIndent(messageData.(map[string]interface{}), "", "  ")

	if errr := json.Unmarshal(rawJSON, &searchResult); errr != nil {
		err = fmt.Errorf("failed to unmarshal result, error: %v", errr)
		return
	}

	messages = make([]MessageDocument, len(searchResult.Hits.Hits))

	for i, hit := range searchResult.Hits.Hits {
		messages[i] = hit.Source
	}

	return
}

func (m *MessageDocument) UpdateMessageUsername(logger *utility.Logger, mu *sync.Mutex) error {
	mu.Lock()
	defer mu.Unlock()

	payload := map[string]interface{}{
		"script": map[string]interface{}{
			"source": "ctx._source.username = params.new_username",
			"lang":   "painless",
			"params": map[string]interface{}{
				"new_username": m.Username,
			},
		},
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"user_id.keyword": m.UserID,
			},
		},
	}

	err := elastic.UpdateByQueryWithScript(storage.DB.Elastic, payload, MessageIndexName)

	if err != nil {
		logger.Error("An error occurred while updating message index: %v", err)
		return err
	}

	logger.Info("Updated username across message index")
	return nil
}
