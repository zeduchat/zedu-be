package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/utility"
)

var ThreadIndexName = "threads"

type Threads struct {
	ID            string     `gorm:"type:uuid;primary_key" json:"thread_id"`
	ChannelsID    string     `gorm:"type:uuid;index" json:"channels_id"`
	EventName     string     `gorm:"type:varchar(200);index" json:"event_name"`
	Username      string     `gorm:"type:varchar(50);index" json:"username"`
	ActionType    string     `gorm:"type:text;index" json:"action_type"`
	Status        string     `gorm:"type:varchar(200);index" json:"status"`
	CreatedAt     time.Time  `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	Messages      []Message  `gorm:"foreignKey:ThreadID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"messages"`
	MessageCount  int64      `gorm:"type:int;" json:"message_count"`
	LastReply     time.Time  `json:"last_reply"`
	AvatarURL     string     `json:"avatar_url"`
	Type          string     `gorm:"default:thread" json:"type"`
	Content       string     `gorm:"type:text;index" json:"message"`
	ChannelName   string     `json:"channel_name"`
	CurrentStatus string     `json:"current_status"`
	FullName      string     `json:"full_name"`
	Email         string     `json:"email"`
	Reactions     []Reaction `gorm:"foreignKey:ThreadID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"reactions"`
}

type ThreadDocument struct {
	ID            string            `json:"thread_id"`
	ChannelsID    string            `json:"channels_id"`
	EventName     string            `json:"event_name"`
	Username      string            `json:"username"`
	ActionType    string            `json:"action_type"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	MessageCount  int64             `json:"message_count"`
	LastReply     time.Time         `json:"last_reply"`
	AvatarURL     string            `json:"avatar_url"`
	Type          string            `json:"type"`
	Content       string            `json:"message"`
	ChannelName   string            `json:"channel_name"`
	CurrentStatus string            `json:"current_status"`
	FullName      string            `json:"full_name"`
	Email         string            `json:"email"`
	UserId        string            `json:"user_id"`
	Messages      []MessageDocument `json:"messages,omitempty"`
}

var Thread_mapping = map[string]interface{}{

	"mappings": map[string]interface{}{
		"properties": map[string]interface{}{
			"id":          map[string]string{"type": "keyword"},
			"channels_id": map[string]string{"type": "keyword"},
			"user_id":     map[string]string{"type": "keyword"},
			"event_name":  map[string]string{"type": "text"},
			"username":    map[string]string{"type": "keyword"},
			"action_type": map[string]string{"type": "text"},
			"status":      map[string]string{"type": "text"},
			"created_at": map[string]string{
				"type":   "date",
				"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
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
			"messages": map[string]interface{}{
				"type":       "nested",
				"properties": MessageMapping,
			},
		},
	},
}

type Reaction struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	ThreadID  string    `gorm:"type:uuid;index" json:"thread_id"`
	UserID    string    `gorm:"type:uuid;index" json:"user_id"`
	Reaction  string    `gorm:"type:varchar(50);index" json:"reaction"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
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

type CreateThreadMsgReq struct {
	Content    string `json:"content" validate:"required"`
	ChannelsID string `json:"channels_id"`
	UserId     string `json:"user_id"`
	ThreadId   string `json:"thread_id"`
}

type FeedMessageRequest struct {
	ChannelID string `json:"channel_id"`
	FullName  string `json:"full_name"`
	UserName  string `json:"user_name"`
	CreatedAt string `json:"created_at"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Type      string `json:"type"`
	Content   string `json:"message"`
	ThreadId  string `json:"thread_id"`
}

func (t *Threads) GetChannelCountInfo(db *gorm.DB, orgId string, days int) (ChannelCountInfo, []ChannelMetrics, error) {
	var (
		cc                ChannelCountInfo
		channelThreadInfo []ChannelMetrics
	)

	dateCondition := fmt.Sprintf("threads.created_at >= NOW() - INTERVAL '%d days'", days)

	_ = db.Model(&t).
		Joins("JOIN channels ON channels.id = threads.channels_id").
		Where("channels.organisation_id = ? AND threads.status = ? AND "+dateCondition, orgId, "success").
		Count(&cc.TotalSuccessThreads).Error

	_ = db.Model(&t).
		Joins("JOIN channels ON channels.id = threads.channels_id").
		Where("channels.organisation_id = ? AND threads.status = ? AND "+dateCondition, orgId, "error").
		Count(&cc.TotalErrorThreads).Error

	_ = db.Model(&t).
		Joins("JOIN channels ON channels.id = threads.channels_id").
		Where("channels.organisation_id = ? AND threads.current_status = ? AND "+dateCondition, orgId, "completed").
		Count(&cc.TotalResolvedThreads).Error

	_ = db.Model(&t).
		Joins("JOIN channels ON channels.id = threads.channels_id").
		Where("threads.type = 'thread'").
		Where("channels.organisation_id = ? AND "+dateCondition, orgId).
		Count(&cc.TotalThreads).Error

	err := db.Model(&Channels{}).
		Select("channels.id, channels.name AS channel_name, "+
			"COUNT(threads.id) AS thread_count, "+
			"SUM(CASE WHEN threads.status = 'success' THEN 1 ELSE 0 END) AS success_count, "+
			"SUM(CASE WHEN threads.status = 'error' THEN 1 ELSE 0 END) AS error_count, "+
			"SUM(CASE WHEN threads.status = 'other' THEN 1 ELSE 0 END) AS other_count").
		Joins("LEFT JOIN threads ON threads.channels_id = channels.id AND "+dateCondition).
		Where("channels.organisation_id = ? ", orgId).
		Group("channels.id, channels.name").
		Scan(&channelThreadInfo).Error

	if err != nil {
		return cc, nil, err
	}
	return cc, channelThreadInfo, nil
}

type Mentions struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	MessageID string    `gorm:"type:uuid;index" json:"message_id"`
	UserID    string    `gorm:"type:uuid;index" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type UpdateThreadStatus struct {
	Status string `json:"status" validate:"required,oneof=pending completed"`
}

func (t *ThreadDocument) CreateThread(db *storage.Database, logger *utility.Logger) error {

	err := elastic.AddDocument(db.Elastic, ThreadIndexName, t.ID, interface{}(&t), logger)

	if err != nil {
		return err
	}

	threadDocument := ChannelDocument{
		ID:           t.ID,
		Type:         "thread",
		ChannelsID:   t.ChannelsID,
		ThreadID:     t.ID,
		UserID:       "",
		Username:     t.Username,
		Content:      t.Content,
		CreatedAt:    t.CreatedAt.Unix(),
		EventName:    t.EventName,
		ActionType:   t.ActionType,
		Status:       t.Status,
		MessageCount: t.MessageCount,
		AvatarURL:    t.AvatarURL,
	}

	err = tydb.InsertDocument(db.TypeSense, t.ChannelsID, threadDocument)
	if err != nil {
		return errors.New("could not create thread document in Typesense an error occurred: " + err.Error())
	}

	return nil
}

func (c *Threads) UpdateThread(db *gorm.DB, req map[string]interface{}) (*Threads, error) {

	err := elastic.UpdateDocument(storage.DB.Elastic, ThreadIndexName, c.ID, req)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to update thread, err: %v", err))
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

func (t *Threads) GetThreadById(db *gorm.DB, threadID string) (*Threads, error) {

	var (
		threadData interface{}
	)

	err := elastic.SelectByID(storage.DB.Elastic, ThreadIndexName, threadID, &threadData)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to fetch thread records, error: %v", err))
	}

	rawJSON, _ := json.MarshalIndent(threadData.(map[string]interface{}), "", "  ")

	if err := json.Unmarshal(rawJSON, &t); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %v", err)

	}

	return t, nil
}

func (t *Threads) GetAllThreadsByChannelID(c *gin.Context, db *gorm.DB, userId, channelID string) ([]Threads, *elastic.PaginationResponse, error) {
	var (
		threads []Threads
	)

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit

	from := (page - 1) * limit

	// Build the query
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"channels_id.keyword": channelID,
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

	var threadData interface{}

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, ThreadIndexName, query, &threadData, c)

	if err != nil {
		return nil, pagR, errors.New(fmt.Sprintf("failed to fetch thread records, error: %v", err))
	}

	threads, err = UnMarsahlThreadResponse(threadData)
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
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"channels_id.keyword": channelID,
						},
					},
					{
						"term": map[string]interface{}{
							"type.keyword": "thread",
						},
					},
				},
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

	var threadData interface{}

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, ThreadIndexName, query, &threadData, c)

	if err != nil {
		return nil, pagR, errors.New(fmt.Sprintf("failed to fetch thread records, error: %v", err))
	}

	threads, err = UnMarsahlThreadResponse(threadData)

	if err != nil {
		return nil, pagR, err
	}

	return threads, pagR, nil
}

func (t *Threads) GetUserThreadsByOrganization(c *gin.Context, db *gorm.DB, userId, organisationID string) ([]Threads, *elastic.PaginationResponse, error) {
	var (
		threads    []Threads
		channelIDs []string
		threadData interface{}
		threadIDs  []string
	)

	pag := elastic.GetPagination(c)
	page, limit := pag.Page, pag.Limit

	from := (page - 1) * limit

	err := db.Model(&Channels{}).
		Select("channels.id").
		Where("channels.organisation_id = ?", organisationID).
		Find(&channelIDs).Error

	if err != nil {
		return nil, nil, fmt.Errorf("Error fetching channel IDs: %v", err)
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							"channels_id.keyword": channelIDs,
						},
					},
					{
						"term": map[string]interface{}{
							"user_id.keyword": userId,
						},
					},
				},
			},
		},
		"stored_fields": []string{},
		"from":          from,
		"size":          limit,
		"aggs": map[string]interface{}{
			"unique_thread_ids": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "thread_id.keyword",
					"size":  limit,
				},
			},
		},
	}

	pagR, err := elastic.SelectWithPagination(storage.DB.Elastic, "messages", query, &threadData, c)

	if err != nil {
		return nil, pagR, errors.New(fmt.Sprintf("failed to fetch thread records, error: %v", err))
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

	rawJSON, _ := json.MarshalIndent(threadData.(map[string]interface{}), "", "  ")

	if err := json.Unmarshal(rawJSON, &searchResult); err != nil {
		return threads, pagR, fmt.Errorf("failed to decode search response: %v", err)
	}

	// Extract unique thread IDs from the aggregation buckets

	for _, bucket := range searchResult.Aggs.UniqueThreadIDs.Buckets {
		threadIDs = append(threadIDs, bucket.Key)
	}

	// Build the query
	query = map[string]interface{}{
		"query": map[string]interface{}{
			"terms": map[string]interface{}{
				"thread_id.keyword": threadIDs,
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

	pagR, err = elastic.SelectWithPagination(storage.DB.Elastic, ThreadIndexName, query, &threadData, c)

	if err != nil {
		return nil, pagR, errors.New(fmt.Sprintf("failed to fetch thread records, error: %v", err))
	}

	threads, err = UnMarsahlThreadResponse(threadData)

	if err != nil {
		return nil, pagR, err
	}

	return threads, pagR, nil
}

func (t *Threads) GetSingleThreadWithRepliesFull(db *gorm.DB, ChannelID, threadID string) (*Threads, error) {
	var thread Threads

	err := db.Model(&Threads{}).
		Where("threads.id = ?", threadID).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC").Preload("Mentions")
		}).
		Select("threads.*, COUNT(messages.id) as message_count").
		Joins("LEFT JOIN messages ON messages.thread_id = threads.id").
		Group("threads.id").
		First(&thread).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("thread not found")
		}
		return nil, err
	}

	return &thread, nil
}

func (r *Threads) GetSingleThreadWithReplies(db *gorm.DB, c *gin.Context, userID, channelID, ThreadID string) (MessagesResp, postgresql.PaginationResponse, error) {

	var (
		userChannels    UserChannels
		messagesResp    MessagesResp
		ErrNotInChannel = errors.New("user not in channel")
	)

	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	if !exist {
		return messagesResp, postgresql.PaginationResponse{}, ErrNotInChannel
	}

	pagination := postgresql.GetPagination(c)
	query := db.Table("messages").
		Select("messages.content AS message, messages.id, messages.username, messages.created_at, messages.updated_at, messages.edited, profiles.full_name, profiles.avatar_url, users.email").
		Joins("left join profiles on profiles.userid = messages.user_id").
		Joins("left join users on users.id = messages.user_id").
		Where("messages.thread_id = ?", ThreadID)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"messages.created_at",
		"desc",
		pagination,
		&messagesResp,
		nil,
	)
	if err != nil {
		return messagesResp, paginationResponse, err
	}

	return messagesResp, paginationResponse, nil
}

func UnMarsahlThreadResponse(threadData interface{}) (threads []Threads, err error) {

	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source Threads `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	rawJSON, _ := json.MarshalIndent(threadData.(map[string]interface{}), "", "  ")

	if errr := json.Unmarshal(rawJSON, &searchResult); errr != nil {
		err = errors.New(fmt.Sprintf("failed to unmarshal result, error: %v", errr))
		return
	}

	threads = make([]Threads, len(searchResult.Hits.Hits))

	for i, hit := range searchResult.Hits.Hits {
		threads[i] = hit.Source
	}

	return
}
