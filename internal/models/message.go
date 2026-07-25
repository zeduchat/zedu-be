package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

var MessageIndexName = "messages"

type Message struct {
	ID         string         `gorm:"type:uuid;primary_key" json:"id"`
	Content    string         `gorm:"column:content; type:text; not null" json:"content"`
	Msg        string         `gorm:"column:message; type:text; not null" json:"message"`
	ChannelsID string         `gorm:"type:uuid;not null;index" json:"channels_id"`
	UserID     string         `gorm:"type:uuid;not null;index" json:"user_id"`
	Username   string         `gorm:"column:username; type:varchar(100)" json:"username"`
	CreatedAt  time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	ThreadID   uuid.UUID      `gorm:"type:uuid;null;index" json:"thread_id"`
	Mentions   []Mention      `gorm:"-" json:"mentions,omitempty"`
	AvatarURL  string         `json:"avatar_url"`
	UserType   string         `json:"user_type"`
	IsPinned   bool           `json:"is_pinned"`
	IsSaved    bool           `gorm:"type:bool;default:false" json:"is_saved"`
	Edited     bool           `gorm:"column:edited;type:bool;default:false" json:"edited,omitempty"`
	EditedAt   *time.Time     `gorm:"column:edited_at" json:"edited_at,omitempty"`
}

type MessageDocument struct {
	ID               string            `json:"id,omitempty"`
	ProfileID        string            `json:"profile_id,omitempty"`
	Content          string            `json:"message"`
	OrganisationID   string            `json:"org_id"`
	ChannelsID       string            `json:"channels_id"`
	UserID           string            `json:"user_id"`
	Username         string            `json:"username"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	DeletedAt        gorm.DeletedAt    `json:"-"`
	AgentMessage     bool              `json:"-"`
	UserType         string            `json:"user_type"`
	ThreadID         uuid.UUID         `json:"thread_id"`
	AvatarURL        string            `json:"avatar_url"`
	DefaultAvatarURL string            `json:"default_avatar_url"`
	Edited           bool              `json:"edited"`
	FullName         string            `json:"full_name"`
	Email            string            `json:"email"`
	Media            []File            `json:"media,omitempty"`
	IsPinned         bool              `json:"is_pinned"`
	IsSaved          bool              `json:"is_saved"`
	Mentions         []Mention         `json:"mentions,omitempty"`
	PinnedDetails    PinnedDetails     `json:"pinned_details,omitempty"`
	Reactions        []ReactionDetails `json:"reactions"`
}

var MessageMapping = map[string]any{
	"id":                 map[string]string{"type": "keyword"},
	"profile_id":         map[string]string{"type": "keyword"},
	"channels_id":        map[string]string{"type": "keyword"},
	"user_id":            map[string]string{"type": "keyword"},
	"org_id":             map[string]string{"type": "keyword"},
	"username":           map[string]string{"type": "keyword"},
	"user_type":          map[string]string{"type": "keyword"},
	"thread_id":          map[string]string{"type": "keyword"},
	"avatar_url":         map[string]string{"type": "text"},
	"default_avatar_url": map[string]string{"type": "text"},
	"edited":             map[string]string{"type": "boolean"},
	"message":            map[string]string{"type": "text"},
	"full_name":          map[string]string{"type": "text"},
	"email":              map[string]string{"type": "text"},
	"created_at": map[string]string{
		"type":   "date",
		"format": "strict_date_optional_time||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
	},
	"media": map[string]any{
		"type":       "nested",
		"properties": MediaMapping,
	},
	"mention": map[string]any{
		"type":       "nested",
		"properties": MentionMapping,
	},
	"updated_at": map[string]string{
		"type":   "date",
		"format": "strict_date_optional_time||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
	},
	"deleted_at": map[string]string{
		"type":   "date",
		"format": "strict_date_optional_time||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
	},
	"is_pinned": map[string]string{
		"type": "boolean",
	},
	"is_saved": map[string]string{
		"type": "boolean",
	},
	"pinned_details": map[string]any{
		"type":       "nested",
		"properties": PinnedDetailsMapping,
	},
	"reactions": map[string]any{
		"type":       "nested",
		"properties": ReactionMapping,
	},
}

var Message_mapping = map[string]any{
	"mappings": map[string]any{
		"properties": MessageMapping,
	},
}

type CreateMessageRequest struct {
	Content    string    `json:"content" validate:"required"`
	UserId     string    `json:"user_id"`
	AgentId    string    `json:"agent_id"`
	ChannelsId string    `json:"channels_id"`
	ThreadId   string    `json:"thread_id" validate:"required"`
	OrgId      string    `json:"org_id"`
	AgentName  string    `json:"agent_name"`
	Media      []File    `json:"media"`
	Mentions   []Mention `json:"mentions"`
}

type EditMessageRequest struct {
	Content    string `json:"content" validate:"required"`
	UserId     string `json:"user_id"`
	ChannelsId string `json:"channels_id"`
	ThreadId   string `json:"thread_id" validate:"required"`
	MessageId  string `json:"message_id" validate:"required"`
	OrgId      string `json:"org_id"`
}

type ForwardThreadMessageRequest struct {
	ThreadId             string     `json:"thread_id" validate:"required"`               //thread id of the message to forward
	ForwardedToChannelId *uuid.UUID `json:"forwarded_to_channel_id" validate:"required"` //channel to forward to
	Content              string     `json:"content"`
	Media                []File     `json:"media"`
	Mentions             []Mention  `json:"mentions"`

	UserId     string `json:"user_id"`
	ChannelsId string `json:"channels_id"` //current channels or DM
}

type ForwardReplyMessageRequest struct {
	ThreadId             string     `json:"thread_id" validate:"required"`
	MessageId            string     `json:"message_id" validate:"required"`
	ForwardedToChannelId *uuid.UUID `json:"forwarded_to_channel_id" validate:"required"` //channel to forward to
	Content              string     `json:"content"`
	Media                []File     `json:"media"`
	Mentions             []Mention  `json:"mentions"`

	UserId     string `json:"user_id"`
	ChannelsId string `json:"channels_id"`
}

func (m *MessageDocument) CreateMessage(db *storage.Database, logger *utility.Logger) (map[string]any, error) {
	var (
		dmChannels   DmChannels
		userChannels UserChannels
		thread       ThreadDocument
	)

	updateResp := map[string]any{}
	previewSect := false

	chanExist := postgresql.CheckExists(db.Postgresql, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db.Postgresql, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || chanExist) && !m.AgentMessage {
		return updateResp, errors.New("user not in channel")
	}

	err := elastic.AddDocument(db.Elastic, MessageIndexName, m.ID, any(&m), logger)
	if err != nil {
		return updateResp, err
	}

	err = thread.GetThreadById(m.ThreadID.String())

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

		req := map[string]any{
			"script": map[string]any{
				"source": script,
				"params": map[string]any{
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

		req := map[string]any{
			"script": map[string]any{
				"source": script,
				"params": map[string]any{
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

	err = thread.GetThreadById(m.ThreadID.String())
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

func (m *Message) UpdateMessage(db *gorm.DB, req map[string]any) (*Message, error) {

	err := elastic.UpdateDocument(storage.DB.Elastic, MessageIndexName, m.ID, req)
	if err != nil {
		return nil, fmt.Errorf("message not found")
	}

	return m, nil
}

func (m *Message) UpdateMessageInPostgres(db *gorm.DB, updates map[string]interface{}) error {
	if err := db.Model(&Message{}).Where("id = ?", m.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update message in postgres: %w", err)
	}
	return nil
}

func (m *Message) UpdateMessageWithScript(db *gorm.DB, req map[string]any) (*Message, error) {

	err := elastic.UpdateDocWithScript(storage.DB.Elastic, MessageIndexName, m.ID, req)
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
		messageData any
	)

	err := elastic.SelectByID(storage.DB.Elastic, MessageIndexName, messageID, &messageData)

	if err != nil {
		return fmt.Errorf("failed to fetch message records, error: %v", err)
	}

	rawJSON, _ := json.MarshalIndent(messageData.(map[string]any), "", "  ")

	if err := json.Unmarshal(rawJSON, &t); err != nil {
		return fmt.Errorf("failed to decode search response: %v", err)

	}

	return nil
}

func (t *MessageDocument) CheckExists() (bool, int, error) {

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"_id": t.ID,
						},
					},
				},
			},
		},
	}

	check, err := elastic.CheckExists(storage.DB.Elastic, MessageIndexName, query)
	if err != nil {
		return false, http.StatusInternalServerError, err
	}

	return check, http.StatusOK, err
}

func (m *MessageDocument) DeleteMessage(db *gorm.DB, logger *utility.Logger) (map[string]any, error) {

	var (
		thread ThreadDocument
	)

	updateResp := map[string]any{}
	previewSect := false

	err := elastic.DeleteDocument(storage.DB.Elastic, MessageIndexName, m.ID)

	if err != nil {
		return updateResp, fmt.Errorf("failed to delete message, err: %v", err)
	}

	err = thread.GetThreadById(m.ThreadID.String())

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

		req := map[string]any{
			"script": map[string]any{
				"source": script,
				"params": map[string]any{
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
		req := map[string]any{
			"script": map[string]any{
				"source": script,
			},
		}

		err = elastic.UpdateDocWithScript(storage.DB.Elastic, ThreadIndexName, m.ThreadID.String(), req)
		if err != nil {
			logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
			return updateResp, err
		}
	}

	err = thread.GetThreadById(m.ThreadID.String())

	if err != nil {
		return updateResp, err
	}

	updateResp["thread_count"] = thread.MessageCount
	updateResp["preview_section"] = previewSect

	return updateResp, nil
}

func (c *Message) DeleteMessageMediaFiles(logger *utility.Logger, db *gorm.DB, mediaFiles []File) (*Message, error) {
	var (
		fileModel File
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

func (t *Message) GetAllMessagesByThreadID(c *gin.Context, ThreadID string) ([]MessageDocument, *elastic.PaginationResponse, error) {
	var (
		messages []MessageDocument
	)

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit

	from := (page - 1) * limit

	// Build the query
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"thread_id": ThreadID,
			},
		},
		"from": from,
		"size": limit,
		"sort": []map[string]any{
			{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
	}

	var messageData any

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, []string{MessageIndexName}, query, &messageData, c)

	if err != nil {
		return nil, pagR, fmt.Errorf("failed to fetch message records, error: %v", err)
	}

	messages, err = UnmarshalMessageResponse(messageData)
	if err != nil {
		return nil, pagR, err
	}

	if storage.DB != nil && storage.DB.Postgresql != nil {
		messages = HydrateMessageProfiles(storage.DB.Postgresql, messages)
	}

	return messages, pagR, nil
}

func UnmarshalMessageResponse(messageData any) (messages []MessageDocument, err error) {

	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source MessageDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	rawJSON, _ := json.MarshalIndent(messageData.(map[string]any), "", "  ")

	if errr := json.Unmarshal(rawJSON, &searchResult); errr != nil {
		err = fmt.Errorf("failed to unmarshal result, error: %v", errr)
		return
	}

	messages = make([]MessageDocument, len(searchResult.Hits.Hits))

	for i, hit := range searchResult.Hits.Hits {
		messages[i] = hit.Source
		messages[i].DefaultAvatarURL = avatar.GenerateDefaultAvatarURL(messages[i].UserID)
	}

	return
}

func HydrateMessageProfiles(db *gorm.DB, messages []MessageDocument) []MessageDocument {
	if len(messages) == 0 || db == nil {
		return messages
	}

	profileIDs := make([]string, 0)
	userOrgKeys := make(map[string]struct{})

	for _, msg := range messages {
		if msg.ProfileID != "" {
			profileIDs = append(profileIDs, msg.ProfileID)
		} else if msg.UserID != "" && msg.UserID != "WEBHOOK" {
			userOrgKeys[fmt.Sprintf("%s:%s", msg.UserID, msg.OrganisationID)] = struct{}{}
		}
	}

	profileMap := make(map[string]Profile)
	userOrgMap := make(map[string]Profile)

	if len(profileIDs) > 0 {
		var profs []Profile
		if err := db.Where("id IN ?", profileIDs).Find(&profs).Error; err == nil {
			for _, p := range profs {
				profileMap[p.ID] = p
			}
		}
	}

	var profModel Profile
	for key := range userOrgKeys {
		parts := strings.Split(key, ":")
		uID, oID := parts[0], parts[1]
		if p, err := profModel.GetOrCreateProfileForOrg(db, uID, oID); err == nil {
			userOrgMap[key] = p
		}
	}

	for i := range messages {
		var p Profile
		var found bool

		if messages[i].ProfileID != "" {
			p, found = profileMap[messages[i].ProfileID]
		}
		if !found && messages[i].UserID != "" && messages[i].UserID != "WEBHOOK" {
			key := fmt.Sprintf("%s:%s", messages[i].UserID, messages[i].OrganisationID)
			p, found = userOrgMap[key]
		}

		if found {
			messages[i].ProfileID = p.ID
			messages[i].Username = p.UserName
			messages[i].FullName = p.FullName
			if p.AvatarURL != "" {
				messages[i].AvatarURL = p.AvatarURL
			}
		}
	}

	return messages
}

func (m *MessageDocument) UpdateMessageUserProfile(logger *utility.Logger, mu *sync.Mutex) error {
	mu.Lock()
	defer mu.Unlock()

	payload := map[string]any{
		"script": map[string]any{
			"source": `
				if (params.new_username != null && !params.new_username.isEmpty()) {
					ctx._source.username = params.new_username;
				}
				if (params.new_avatarurl != null && !params.new_avatarurl.isEmpty()) {
					ctx._source.avatar_url = params.new_avatarurl;
				}
			`,
			"lang": "painless",
			"params": map[string]any{
				"new_username":  m.Username,
				"new_avatarurl": m.AvatarURL,
			},
		},
		"query": map[string]any{
			"term": map[string]any{
				"user_id": m.UserID,
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
