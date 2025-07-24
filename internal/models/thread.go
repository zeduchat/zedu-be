package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

var ThreadIndexName = "threads"

type Threads struct {
	ID            string                 `gorm:"type:uuid;primary_key" json:"thread_id"`
	ChannelsID    string                 `gorm:"type:uuid;index" json:"channels_id"`
	EventName     string                 `gorm:"type:varchar(200);index" json:"event_name,omitempty"`
	Username      string                 `gorm:"type:varchar(50);index" json:"username"`
	ActionType    string                 `gorm:"type:text;index" json:"action_type,omitempty"`
	Status        string                 `gorm:"type:varchar(200);index" json:"status,omitempty"`
	CreatedAt     time.Time              `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	Messages      []Message              `gorm:"foreignKey:ThreadID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"messages"`
	MessageCount  int64                  `gorm:"type:int;" json:"message_count,omitempty"`
	LastReply     time.Time              `json:"last_reply"`
	AvatarURL     string                 `json:"avatar_url"`
	Type          string                 `gorm:"default:thread" json:"type"`
	Content       string                 `gorm:"type:text;index" json:"message"`
	ChannelName   string                 `json:"channel_name,omitempty"`
	CurrentStatus string                 `json:"current_status"`
	FullName      string                 `json:"full_name"`
	Email         string                 `json:"email"`
	Edited        bool                   `json:"edited"`
	IsPinned      bool                   `json:"is_pinned"`
	IsSaved       bool                   `json:"is_saved,omitempty"`
	UserType      string                 `json:"user_type"`
	Count         int                    `json:"frequency,omitempty"`
	UserId        string                 `json:"user_id"`
	Media         []UploadedFileResponse `json:"media,omitempty"`
	Mentions      []Mention              `json:"mentions,omitempty"`
	OrgansationID string                 `json:"org_id,omitempty"`
	State         string                 `json:"state,omitempty"`
	PinnedDetails PinnedDetails          `json:"pinned_details,omitempty"`
	Reactions     []ReactionDetails      `json:"reactions"`
}

type ThreadDocument struct {
	ID            string                 `json:"thread_id"`
	ChannelsID    string                 `json:"channels_id"`
	OrgansationID string                 `json:"org_id"`
	EventName     string                 `json:"event_name,omitempty"`
	Username      string                 `json:"username"`
	ActionType    string                 `json:"action_type,omitempty"`
	Status        string                 `json:"status,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	MessageCount  int64                  `json:"message_count"`
	LastReply     time.Time              `json:"last_reply"`
	AvatarURL     string                 `json:"avatar_url"`
	UserType      string                 `json:"user_type"`
	Type          string                 `json:"type"`
	Content       string                 `json:"message"`
	ChannelName   string                 `json:"channel_name,omitempty"`
	CurrentStatus string                 `json:"current_status"`
	FullName      string                 `json:"full_name"`
	Email         string                 `json:"email"`
	UserId        string                 `json:"user_id"`
	Edited        bool                   `json:"edited"`
	IsPinned      bool                   `json:"is_pinned"`
	Messages      []MessageDocument      `json:"messages,omitempty"`
	Count         int                    `json:"frequency,omitempty"`
	Media         []UploadedFileResponse `json:"media,omitempty"`
	Mentions      []Mention              `json:"mentions,omitempty"`
	State         string                 `json:"state,omitempty"`
	PinnedDetails PinnedDetails          `json:"pinned_details,omitempty"`
	Reactions     []ReactionDetails      `json:"reactions"`
}

var MediaMapping = map[string]any{
	"mappings": map[string]any{
		"properties": map[string]any{
			"id":        map[string]string{"type": "text"},
			"file_name": map[string]string{"type": "keyword"},
			"file_type": map[string]string{"type": "text"},
			"file_link": map[string]string{"type": "text"},
		},
	},
}

var MentionMapping = map[string]any{
	"mappings": map[string]any{
		"properties": map[string]any{
			"id":   map[string]string{"type": "text"},
			"type": map[string]string{"type": "text"},
		},
	},
}

var PinnedDetailsMapping = map[string]any{
	"mappings": map[string]any{
		"properties": map[string]any{
			"username": map[string]string{"type": "keyword"},
			"email":    map[string]string{"type": "keyword"},
		},
	},
}

var Thread_mapping = map[string]any{
	"mappings": map[string]any{
		"properties": map[string]any{
			"id":          map[string]string{"type": "keyword"},
			"channels_id": map[string]string{"type": "keyword"},
			"user_id":     map[string]string{"type": "keyword"},
			"org_id":      map[string]string{"type": "keyword"},
			"edited":      map[string]string{"type": "boolean"},
			"event_name":  map[string]string{"type": "text"},
			"username":    map[string]string{"type": "keyword"},
			"user_type":   map[string]string{"type": "keyword"},
			"action_type": map[string]string{"type": "text"},
			"status":      map[string]string{"type": "text"},
			"state":       map[string]string{"type": "text"},
			"created_at": map[string]string{
				"type": "date",
				// "format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
				"format": "strict_date_optional_time||epoch_millis",
			},
			"message_count": map[string]string{"type": "integer"},
			"last_reply": map[string]string{
				"type":   "date",
				"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
			},
			"avatar_url":     map[string]string{"type": "text"},
			"type":           map[string]string{"type": "keyword"},
			"message":        map[string]string{"type": "text"},
			"channel_name":   map[string]string{"type": "keyword"},
			"current_status": map[string]string{"type": "text"},
			"full_name":      map[string]string{"type": "text"},
			"email":          map[string]string{"type": "text"},
			"media": map[string]any{
				"type":       "nested",
				"properties": MediaMapping,
			},
			"mentions": map[string]any{
				"type":       "nested",
				"properties": MentionMapping,
			},
			"messages": map[string]any{
				"type":       "nested",
				"properties": MessageMapping,
			},
			"is_pinned": map[string]string{
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
		},
	},
}

type ChannelDocument struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	ChannelsID   string `json:"channels_id"`
	ThreadID     string `json:"thread_id"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Content      string `json:"content"`
	CreatedAt    int64  `json:"created_at"`
	EventName    string `json:"event_name"`
	ActionType   string `json:"action_type"`
	Status       string `json:"status"`
	MessageCount int64  `json:"message_count"`
	AvatarURL    string `json:"avatar_url"`
}

type ChannelCountInfo struct {
	TotalSuccessThreads  int64 `json:"total_success_threads"`
	TotalErrorThreads    int64 `json:"total_error_threads"`
	TotalThreads         int64 `json:"total_threads"`
	TotalResolvedThreads int64 `json:"total_resolved_threads"`
}

type ChannelMetrics struct {
	ChannelName  string `json:"channel_name"`
	ThreadCount  int64  `json:"thread_count"`
	SuccessCount int64  `json:"success_count"`
	ErrorCount   int64  `json:"error_count"`
	OtherCount   int64  `json:"other_count"`
}

type Mention struct {
	Type string `json:"type" validate:"required,oneof=user bot"`
	ID   string `json:"id"`
}

type CreateThreadMsgReq struct {
	Content    string                 `json:"content" validate:"required"`
	Media      []UploadedFileResponse `json:"media"`
	Mentions   []Mention              `json:"mentions"`
	ChannelsID string                 `json:"channels_id"`
	Message    string                 `json:"message"`
	UserId     string                 `json:"user_id"`
	ThreadId   string                 `json:"thread_id"`
	OrgId      string                 `json:"org_id"`
	AgentName  string                 `json:"agent_name"`
	Type       string                 `json:"type"`
}

type BotReturnRequest struct {
	ChannelID      string                 `json:"channel_id"`
	Content        string                 `json:"message"`
	Media          []UploadedFileResponse `json:"media"`
	State          string                 `json:"state"`
	Mentions       []Mention              `json:"mentions"`
	OperationPrice *float64               `json:"operation_price"`
}

type FeedMessageRequest struct {
	ChannelID   string                 `json:"channel_id"`
	FullName    string                 `json:"full_name"`
	UserName    string                 `json:"username"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	Email       string                 `json:"email"`
	AvatarURL   string                 `json:"avatar_url,omitempty"`
	MessageId   string                 `json:"message_id,omitempty"`
	Type        string                 `json:"type"`
	Content     string                 `json:"message"`
	ThreadId    string                 `json:"thread_id"`
	OrgId       string                 `json:"org_id"`
	UserId      string                 `json:"user_id"`
	Media       []UploadedFileResponse `json:"media"`
	UserType    string                 `json:"user_type"`
	Id          string                 `json:"id,omitempty"`
	State       string                 `json:"state"`
	ChannelName string                 `json:"channel_name,omitempty"`
}

type Mentions struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	MessageID string    `gorm:"type:uuid;index" json:"message_id"`
	UserID    string    `gorm:"type:uuid;index" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type TriggerTickRequest struct {
	ChannelID      string `gorm:"type:uuid" json:"channel_id" validate:"required"`
	OrganisationID string `gorm:"type:uuid" json:"organisation_id" validate:"required"`
}

type UpdateThreadStatus struct {
	Status string `json:"status" validate:"required,oneof=pending completed"`
}

type UpdateThreadMessage struct {
	Message   string `json:"content" validate:"required"`
	ThreadId  string `json:"thread_id"`
	ChannelId string `json:"channel_id"`
}

func (t *Threads) GetChannelCountInfo(db *storage.Database, orgId string, days int) (ChannelCountInfo, []ChannelMetrics, error) {
	var (
		CC         ChannelCountInfo
		CTI        []ChannelMetrics
		org        Organisation
		startTime  = time.Now().AddDate(0, 0, -days).Format(time.RFC3339)
		endTime    = time.Now().Format(time.RFC3339)
		channelIDs = make([]string, 0)
	)

	exists := postgresql.CheckExists(db.Postgresql, &org, "id = ?", orgId)
	if !exists {
		return CC, nil, fmt.Errorf("organisation does not exist")
	}

	err := db.Postgresql.Model(&Channels{}).
		Select("channels.id").
		Where("channels.organisation_id = ?", orgId).
		Find(&channelIDs).Error

	if err != nil {
		return CC, nil, fmt.Errorf("error fetching channel IDs: %v", err)
	}

	//eliminate
	if len(channelIDs) == 0 {
		return CC, nil, fmt.Errorf("no channels found for the organisation")
	}

	query := map[string]any{
		"size": 0,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"terms": map[string]any{
							"channels_id.keyword": channelIDs,
						},
					},
				},
			},
		},
		"aggs": map[string]any{
			"total_success_threads": map[string]any{
				"filter": map[string]any{
					"bool": map[string]any{
						"must": []map[string]any{
							{
								"term": map[string]any{
									"status.keyword": "success",
								},
							},
						},
						"filter": []map[string]any{
							{
								"range": map[string]any{
									"created_at": map[string]any{
										"gte": startTime,
										"lte": endTime,
									},
								},
							},
						},
					},
				},
			},
			"total_error_threads": map[string]any{
				"filter": map[string]any{
					"bool": map[string]any{
						"must": []map[string]any{
							{
								"term": map[string]any{
									"status.keyword": "error",
								},
							},
						},
						"filter": []map[string]any{
							{
								"range": map[string]any{
									"created_at": map[string]any{
										"gte": "now-4d/d",
										"lte": "now/d",
									},
								},
							},
						},
					},
				},
			},
			"total_resolved_threads": map[string]any{
				"filter": map[string]any{
					"bool": map[string]any{
						"must": []map[string]any{
							{
								"term": map[string]any{
									"current_status.keyword": "completed",
								},
							},
						},
						"filter": []map[string]any{
							{
								"range": map[string]any{
									"created_at": map[string]any{
										"gte": startTime,
										"lte": endTime,
									},
								},
							},
						},
					},
				},
			},
			"total_threads": map[string]any{
				"filter": map[string]any{
					"bool": map[string]any{
						"must": []map[string]any{
							{
								"term": map[string]any{
									"type.keyword": "thread",
								},
							},
						},
						"filter": []map[string]any{
							{
								"range": map[string]any{
									"created_at": map[string]any{
										"gte": startTime,
										"lte": endTime,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var InfoCount any
	err = elastic.SelectAll(db.Elastic, ThreadIndexName, query, &InfoCount)
	if err != nil {
		return CC, nil, err
	}

	// Extract the counts from the aggregation buckets
	InfoCountMap, ok := InfoCount.(map[string]any)
	if !ok {
		return CC, nil, fmt.Errorf("failed to extract counts from aggregation buckets")
	}

	CC.TotalSuccessThreads = int64(InfoCountMap["aggregations"].(map[string]any)["total_success_threads"].(map[string]any)["doc_count"].(float64))
	CC.TotalErrorThreads = int64(InfoCountMap["aggregations"].(map[string]any)["total_error_threads"].(map[string]any)["doc_count"].(float64))
	CC.TotalResolvedThreads = int64(InfoCountMap["aggregations"].(map[string]any)["total_resolved_threads"].(map[string]any)["doc_count"].(float64))
	CC.TotalThreads = int64(InfoCountMap["aggregations"].(map[string]any)["total_threads"].(map[string]any)["doc_count"].(float64))

	//query for aggregating metrics per channel
	query = map[string]any{
		"size": 0,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"terms": map[string]any{
							"channels_id.keyword": channelIDs,
						},
					},
					{
						"range": map[string]any{
							"created_at": map[string]any{
								"gte": startTime,
								"lte": endTime,
							},
						},
					},
				},
			},
		},
		"aggs": map[string]any{
			"channels": map[string]any{
				"terms": map[string]any{
					"field": "channels_id.keyword",
				},
				"aggs": map[string]any{
					"channel_name": map[string]any{
						"terms": map[string]any{
							"field": "channel_name.keyword",
						},
					},
					"thread_count": map[string]any{
						"value_count": map[string]any{
							"field": "thread_id.keyword",
						},
					},
					"success_count": map[string]any{
						"filter": map[string]any{
							"bool": map[string]any{
								"must": []map[string]any{
									{
										"term": map[string]any{
											"status.keyword": "success",
										},
									},
								},
							},
						},
					},
					"error_count": map[string]any{
						"filter": map[string]any{
							"bool": map[string]any{
								"must": []map[string]any{
									{
										"term": map[string]any{
											"status.keyword": "error",
										},
									},
								},
							},
						},
					},
					"other_count": map[string]any{
						"filter": map[string]any{
							"bool": map[string]any{
								"must": []map[string]any{
									{
										"term": map[string]any{
											"status.keyword": "other",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var ChannelInfoCount any
	err = elastic.SelectAll(db.Elastic, ThreadIndexName, query, &ChannelInfoCount)
	if err != nil {
		return CC, nil, err
	}

	// Extract the counts from the aggregation buckets
	ChannelInfoCountMap, ok := ChannelInfoCount.(map[string]any)
	if !ok {
		return CC, nil, fmt.Errorf("failed to extract counts from aggregation buckets")
	}

	for _, bucket := range ChannelInfoCountMap["aggregations"].(map[string]any)["channels"].(map[string]any)["buckets"].([]any) {
		bucketMap := bucket.(map[string]any)
		channelMetrics := ChannelMetrics{
			ChannelName:  bucketMap["channel_name"].(map[string]any)["buckets"].([]any)[0].(map[string]any)["key"].(string),
			ThreadCount:  int64(bucketMap["thread_count"].(map[string]any)["value"].(float64)),
			SuccessCount: int64(bucketMap["success_count"].(map[string]any)["doc_count"].(float64)),
			ErrorCount:   int64(bucketMap["error_count"].(map[string]any)["doc_count"].(float64)),
			OtherCount:   int64(bucketMap["other_count"].(map[string]any)["doc_count"].(float64)),
		}
		CTI = append(CTI, channelMetrics)
	}

	return CC, CTI, nil
}

func (t *ThreadDocument) CreateThread(db *storage.Database, logger *utility.Logger) error {
	err := elastic.AddDocument(db.Elastic, ThreadIndexName, t.ID, any(&t), logger)

	if err != nil {
		return err
	}

	return nil
}

func (c *Threads) UpdateThread(db *gorm.DB, req map[string]any) (*Threads, error) {

	err := elastic.UpdateDocument(storage.DB.Elastic, ThreadIndexName, c.ID, req)

	if err != nil {
		return nil, fmt.Errorf("thread not found")
	}

	return c, nil
}

func (c *Threads) UpdateThreadWithScript(db *gorm.DB, req map[string]any) (*Threads, error) {

	err := elastic.UpdateDocWithScript(storage.DB.Elastic, ThreadIndexName, c.ID, req)

	if err != nil {
		return nil, fmt.Errorf("thread not found")
	}

	return c, nil
}

func (c *Threads) DeleteThreadMediaFiles(logger *utility.Logger, db *gorm.DB, mediaFiles []UploadedFileResponse) (*Threads, error) {
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

func (c *Threads) DeleteThread(db *gorm.DB) (*Threads, error) {
	messageQuery := map[string]any{
		"query": map[string]any{
			"match": map[string]any{
				"thread_id": c.ID,
			},
		},
	}

	//deletes messages(replies to a thread)
	err := elastic.DeleteByQuery(storage.DB.Elastic, MessageIndexName, messageQuery)

	if err != nil {
		return nil, fmt.Errorf("failed to delete thread messages, err: %v", err)
	}

	//deletes thread
	err = elastic.DeleteDocument(storage.DB.Elastic, ThreadIndexName, c.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid thread uuid supplied: %v", err)
	}

	return c, nil
}

func (c *Threads) ClearDMThreadsByChannelID(db *gorm.DB) (*Threads, error) {

	query := map[string]any{
		"query": map[string]any{
			"match": map[string]any{
				"channels_id": c.ID,
			},
		},
	}

	err := elastic.DeleteByQuery(storage.DB.Elastic, MessageIndexName, query)
	if err != nil {
		return nil, fmt.Errorf("failed to delete channel messages, err: %v", err)
	}

	err = elastic.DeleteByQuery(storage.DB.Elastic, ThreadIndexName, query)
	if err != nil {
		return nil, fmt.Errorf("invalid channel uuid supplied: %v", err)
	}

	return c, nil
}

func (m *Mentions) CreateMention(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, m)
	if err != nil {
		return err
	}
	return nil
}

func (t *ThreadDocument) GetThreadById(db *gorm.DB, threadID string) error {

	var (
		threadData any
	)

	err := elastic.SelectByID(storage.DB.Elastic, ThreadIndexName, threadID, &threadData)

	if err != nil {
		return fmt.Errorf("failed to fetch thread records, error: %v", err)
	}

	rawJSON, _ := json.MarshalIndent(threadData.(map[string]any), "", "  ")

	if err := json.Unmarshal(rawJSON, &t); err != nil {
		return fmt.Errorf("failed to decode search response: %v", err)

	}

	return nil
}

func (t *ThreadDocument) CheckUserThreadExists() (bool, int, error) {

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id.keyword": t.ChannelsID,
						},
					},
					{
						"term": map[string]any{
							"user_id.keyword": t.UserId,
						},
					},
				},
			},
		},
	}

	check, err := elastic.CheckExists(storage.DB.Elastic, ThreadIndexName, query)
	if err != nil {
		return false, http.StatusInternalServerError, err
	}

	return check, http.StatusOK, err
}

func (t *ThreadDocument) CheckExists() (bool, int, error) {

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

	check, err := elastic.CheckExists(storage.DB.Elastic, ThreadIndexName, query)
	if err != nil {
		return false, http.StatusInternalServerError, err
	}

	return check, http.StatusOK, err
}

func (t *Threads) GetAllThreadsByChannelID(c *gin.Context, db *gorm.DB, userId, channelID string) ([]Threads, *elastic.PaginationResponse, error) {
	var (
		threads     []Threads
		channel     Channels
		userChannel UserChannels
		dmChannel   DmChannels
	)

	chanExist, _ := channel.CheckChannelExists(db, channelID)
	dmChanExist, _ := dmChannel.CheckChannelExists(db, channelID, userId)

	if !(dmChanExist || chanExist) {
		return nil, &elastic.PaginationResponse{}, errors.New("channel does not exist")
	}

	if chanExist {
		userExist := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", channelID, userId)

		if channel.IsPrivate && !userExist {
			return nil, &elastic.PaginationResponse{}, errors.New("permission denied, private channel")
		}
	}

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit

	from := (page - 1) * limit

	// Build the query
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"channels_id.keyword": channelID,
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

	var threadData any

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, ThreadIndexName, query, &threadData, c)

	if err != nil {
		return nil, pagR, fmt.Errorf("failed to fetch thread records, error: %v", err)
	}

	threads, err = UnmarshalThreadResponse(threadData)
	if err != nil {
		return nil, pagR, err
	}

	return threads, pagR, nil
}

func (t *Threads) GetThreadsByChannelID(c *gin.Context, db *gorm.DB, userId, channelID string) ([]Threads, *elastic.PaginationResponse, error) {
	var (
		threads []Threads
	)

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit

	from := (page - 1) * limit

	// Build the query
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id.keyword": channelID,
						},
					},
					{
						"term": map[string]any{
							"type.keyword": "thread",
						},
					},
				},
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

	var threadData any

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, ThreadIndexName, query, &threadData, c)

	if err != nil {
		return nil, pagR, fmt.Errorf("failed to fetch thread records, error: %v", err)
	}

	threads, err = UnmarshalThreadResponse(threadData)

	if err != nil {
		return nil, pagR, err
	}

	return threads, pagR, nil
}

func (t *Threads) GetUserThreadsByOrganization(c *gin.Context, db *gorm.DB, userId, organisationID string) ([]Threads, *elastic.PaginationResponse, error) {
	var (
		threads    []Threads
		channelIDs []string
		threadData any
		threadIDs  []string
		org        Organisation
	)

	threads = make([]Threads, 0)

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit

	from := (page - 1) * limit

	exists := postgresql.CheckExists(db, &org, "id = ?", organisationID)
	if !exists {
		return nil, nil, fmt.Errorf("organisation does not exist")
	}

	err := db.Model(&Channels{}).
		Select("channels.id").
		Where("channels.organisation_id = ?", organisationID).
		Find(&channelIDs).Error

	if err != nil {
		return nil, nil, fmt.Errorf("error fetching channel IDs: %v", err)
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"terms": map[string]any{
							"channels_id.keyword": channelIDs,
						},
					},
					{
						"term": map[string]any{
							"user_id.keyword": userId,
						},
					},
				},
			},
		},
		"stored_fields": []string{},
		"from":          from,
		"size":          limit,
		"aggs": map[string]any{
			"unique_thread_ids": map[string]any{
				"terms": map[string]any{
					"field": "thread_id.keyword",
					"size":  limit,
				},
			},
		},
	}

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, "messages", query, &threadData, c)

	if err != nil {
		return nil, pagR, fmt.Errorf("failed to fetch thread records, error in %v", err)
	}

	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source Message `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Aggs struct {
			UniqueThreadIDs struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"unique_thread_ids"`
		} `json:"aggregations"`
	}

	rawJSON, _ := json.MarshalIndent(threadData.(map[string]any), "", "  ")

	if err := json.Unmarshal(rawJSON, &searchResult); err != nil {
		return threads, pagR, fmt.Errorf("failed to decode search response: %v", err)
	}

	// Extract unique thread IDs from the aggregation buckets
	threadIDs = make([]string, 0)

	for _, bucket := range searchResult.Aggs.UniqueThreadIDs.Buckets {
		threadIDs = append(threadIDs, bucket.Key)
	}

	if len(threadIDs) == 0 {
		return threads, pagR, nil
	}

	// Build the query
	query = map[string]any{
		"query": map[string]any{
			"terms": map[string]any{
				"thread_id.keyword": threadIDs,
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

	pagR, err = elastic.SelectWithPagination(storage.DB.Elastic, ThreadIndexName, query, &threadData, c)

	if err != nil {
		return nil, pagR, fmt.Errorf("failed to fetch thread records, error: %v", err)
	}

	threads, err = UnmarshalThreadResponse(threadData)

	if err != nil {
		return nil, pagR, err
	}

	return threads, pagR, nil
}

func UnmarshalThreadResponse(threadData any) (threads []Threads, err error) {

	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source Threads `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	rawJSON, _ := json.MarshalIndent(threadData.(map[string]any), "", "  ")

	if errr := json.Unmarshal(rawJSON, &searchResult); errr != nil {
		err = fmt.Errorf("failed to unmarshal result, error: %v", errr)
		return
	}

	threads = make([]Threads, len(searchResult.Hits.Hits))

	for i, hit := range searchResult.Hits.Hits {
		threads[i] = hit.Source
	}

	return
}

func (t *Threads) GetAllGroupThreadsByChannelID(c *gin.Context, db *gorm.DB, channelID string, timeRange time.Time) ([]Threads, *elastic.PaginationResponse, error) {
	var (
		threads []Threads
	)

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit
	threads = make([]Threads, 0)

	pagR := &elastic.PaginationResponse{
		PageCount:       0,
		CurrentPage:     page,
		TotalPagesCount: 0,
	}

	// Build the query
	cardinality_query := map[string]any{
		"size": 0,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"range": map[string]any{
							"created_at": map[string]any{
								"gte": timeRange.Format(time.RFC3339),
								"lte": time.Now().Format(time.RFC3339),
							},
						},
					},
					{
						"term": map[string]any{
							"type.keyword": "thread",
						},
					},
					{
						"term": map[string]any{
							"channels_id.keyword": channelID,
						},
					},
				},
			},
		},
		"aggs": map[string]any{
			"unique_threads": map[string]any{
				"cardinality": map[string]any{
					"field": "message.keyword",
				},
			},
		},
	}

	var cardData any

	err := elastic.SelectAll(storage.DB.Elastic, ThreadIndexName, cardinality_query, &cardData)
	if err != nil {
		return nil, pagR, fmt.Errorf("failed to fetch number of group thread records, error: %v", err)
	}

	var cardinalityResult struct {
		Aggregations struct {
			UniqueThreads struct {
				Value int `json:"value"`
			} `json:"unique_threads"`
		} `json:"aggregations"`
	}

	rawJSON, _ := json.MarshalIndent(cardData.(map[string]any), "", "  ")

	if errr := json.Unmarshal(rawJSON, &cardinalityResult); errr != nil {
		err = fmt.Errorf("failed to unmarshal result, error: %v", errr)
		return nil, pagR, fmt.Errorf("failed to fetch number of group thread records, error: %v", err)
	}

	// Total unique threads and compute partitions
	totalThreads := cardinalityResult.Aggregations.UniqueThreads.Value
	numPartitions := int(math.Ceil(float64(totalThreads) / float64(limit)))

	if page-1 >= numPartitions {
		return threads, pagR, nil
	}

	pagR.TotalPagesCount = numPartitions
	pagR.SummaryCount = totalThreads

	paginatedQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"range": map[string]any{
							"created_at": map[string]any{
								"gte": timeRange.Format(time.RFC3339),
								"lte": time.Now().Format(time.RFC3339),
							},
						},
					},
					{
						"term": map[string]any{
							"type.keyword": "thread",
						},
					},
					{
						"term": map[string]any{
							"channels_id.keyword": channelID,
						},
					},
				},
			},
		},
		"aggs": map[string]any{
			"partitioned_threads": map[string]any{
				"terms": map[string]any{
					"field": "message.keyword",
					"size":  page * limit,
					"order": map[string]any{
						"_count": "desc",
					},
				},
				"aggs": map[string]any{
					"top_thread_hits": map[string]any{
						"top_hits": map[string]any{
							"size": 1,
						},
					},
				},
			},
		},
	}

	var threadData any

	err = elastic.SelectAll(storage.DB.Elastic, ThreadIndexName, paginatedQuery, &threadData)

	if err != nil {
		return nil, pagR, fmt.Errorf("failed to fetch group thread records, error: %v", err)
	}

	var paginatedResult struct {
		Aggregations struct {
			PartitionedThreads struct {
				Buckets []struct {
					Key           string `json:"key"`
					DocCount      int    `json:"doc_count"`
					TopThreadHits struct {
						Hits struct {
							Hits []struct {
								Source Threads `json:"_source"`
							} `json:"hits"`
						} `json:"hits"`
					} `json:"top_thread_hits"`
				} `json:"buckets"`
			} `json:"partitioned_threads"`
		} `json:"aggregations"`
	}

	rawJSON, _ = json.MarshalIndent(threadData.(map[string]any), "", "  ")

	if errr := json.Unmarshal(rawJSON, &paginatedResult); errr != nil {
		err = fmt.Errorf("failed to unmarshal result, error: %v", errr)
		return nil, pagR, fmt.Errorf("failed to unmarshal group thread records, error: %v", err)
	}

	result := paginatedResult.Aggregations.PartitionedThreads.Buckets

	if len(result) > limit {
		result = result[limit:]
	}

	threads = make([]Threads, len(result))

	for ind, bucket := range result {
		threads[ind] = bucket.TopThreadHits.Hits.Hits[0].Source
		threads[ind].Count = bucket.DocCount
	}

	return threads, pagR, nil
}

func (t *ThreadDocument) UpdateThreadUserProfile(logger *utility.Logger, mu *sync.Mutex) {

	mu.Lock()
	defer mu.Unlock()

	script := `
	if (ctx._source.containsKey("messages")) {
		for (int i = 0; i < ctx._source.messages.size(); i++) {
			if (ctx._source.messages[i].user_id == params.user_id) {
				if (params.new_username != null && !params.new_username.isEmpty()) {
					ctx._source.messages[i].username = params.new_username;
				}
				if (params.new_avatarurl != null && !params.new_avatarurl.isEmpty()) {
					ctx._source.messages[i].avatar_url = params.new_avatarurl;
				}
			}
		}
	}
	if (ctx._source.user_id == params.user_id) {
		if (params.new_username != null && !params.new_username.isEmpty()) {
			ctx._source.username = params.new_username;
		}
		if (params.new_avatarurl != null && !params.new_avatarurl.isEmpty()) {
			ctx._source.avatar_url = params.new_avatarurl;
		}
	}
	`

	req := map[string]any{
		"script": map[string]any{
			"source": script,
			"lang":   "painless",
			"params": map[string]any{
				"user_id":       t.UserId,
				"new_username":  t.Username,
				"new_avatarurl": t.AvatarURL,
			},
		},
		"query": map[string]any{
			"term": map[string]any{
				"user_id.keyword": t.UserId,
			},
		},
	}

	err := elastic.UpdateByQueryWithScript(storage.DB.Elastic, req, ThreadIndexName)

	if err != nil {
		logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
	}

	logger.Info("Updated username across thread index")
}
