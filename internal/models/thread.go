package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
)

type Threads struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"thread_id"`
	ChannelsID    string    `gorm:"type:uuid;index" json:"channels_id"`
	EventName     string    `gorm:"type:varchar(200);index" json:"event_name"`
	Username      string    `gorm:"type:varchar(50);index" json:"username"`
	ActionType    string    `gorm:"type:text;index" json:"action_type"`
	Status        string    `gorm:"type:varchar(200);index" json:"status"`
	CreatedAt     time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	Messages      []Message `gorm:"foreignKey:ThreadID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"messages"`
	MessageCount  int64     `gorm:"type:int;" json:"message_count"`
	LastReply     time.Time `json:"last_reply"`
	AvatarURL     string    `json:"avatar_url"`
	Type          string    `gorm:"default:thread" json:"type"`
	Content       string    `gorm:"type:text;index" json:"content"`
	ChannelName   string    `json:"channel_name"`
	CurrentStatus string    `json:"current_status"`
	FullName      string    `json:"full_name"`
	Email         string    `json:"email"`
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
	TotalThreads         int64 `json:"total_threads"`
	TotalErrorThreads    int64 `json:"total_error_threads"`
	TotalMembers         int64 `json:"total_members"`
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
}

func (t *Threads) GetChannelCountInfo(db *gorm.DB, orgId string, days int) (ChannelCountInfo, []ChannelMetrics, error) {
	var (
		cc ChannelCountInfo
		om OrgUserManagement
	)

	// Determine the date condition based on the range
	dateCondition := fmt.Sprintf("created_at >= NOW() - INTERVAL '%d days'", days)

	// Channel count info
	_ = db.Model(&t).
		Joins("JOIN channels ON channels.id = threads.channels_id").
		Where("channels.organisation_id = ? AND "+dateCondition, orgId).
		Count(&cc.TotalThreads).Error

	_ = db.Model(&t).
		Joins("JOIN channels ON channels.id = threads.channels_id").
		Where("channels.organisation_id = ? AND threads.status = ? AND "+dateCondition, orgId, "failed").
		Count(&cc.TotalErrorThreads).Error

	_ = db.Model(&t).
		Joins("JOIN channels ON channels.id = threads.channels_id").
		Where("channels.organisation_id = ? AND threads.status = ? AND "+dateCondition, orgId, "success").
		Count(&cc.TotalResolvedThreads).Error

	_ = db.Model(&om).
		Where("organisation_id = ?", orgId).
		Count(&cc.TotalMembers).Error

	// Channel metrics
	var channels []Channels
	var channelThreadInfo []ChannelMetrics

	_ = postgresql.SelectAllFromDb(db, "", &channels, "organisation_id = ?", orgId)

	for _, channel := range channels {
		cm, _ := t.ChannelMetrics(db, channel, dateCondition)
		channelThreadInfo = append(channelThreadInfo, cm)
	}
	return cc, channelThreadInfo, nil
}


func (t *Threads) ChannelMetrics(db *gorm.DB, channel Channels, dateCondition string) (ChannelMetrics, error) {
	var (
		cm ChannelMetrics
	)

	cm.ChannelName = channel.Name
	_ = db.Where("channels_id = ? AND "+dateCondition, channel.ID).Model(&t).Count(&cm.ThreadCount).Error
	_ = db.Where("channels_id = ? AND status = ? AND "+dateCondition, channel.ID, "success").Model(&t).Count(&cm.SuccessCount).Error
	_ = db.Where("channels_id = ? AND status = ? AND "+dateCondition, channel.ID, "failed").Model(&t).Count(&cm.ErrorCount).Error
	_ = db.Where("channels_id = ? AND status = ? AND "+dateCondition, channel.ID, "other").Model(&t).Count(&cm.OtherCount).Error
	return cm, nil
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

func (t *Threads) CreateThread(db *gorm.DB, typesenseDb *typesense.Client) error {

	err := postgresql.CreateOneRecord(db, t)
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

	err = tydb.InsertDocument(typesenseDb, t.ChannelsID, threadDocument)
	if err != nil {
		return errors.New("could not create thread document in Typesense")
	}

	return nil
}

func (c *Threads) UpdateThread(db *gorm.DB) (*Threads, error) {
	result, err := postgresql.SaveAllFields(db, &c)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update thread")
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

func (t *Threads) GetThreadById(db *gorm.DB, ChannelID, threadID string) (*Threads, error) {

	err, nerr := postgresql.SelectOneFromDb(db, &t, "id = ? and channels_id = ?", threadID, ChannelID)
	if nerr != nil {
		return t, err
	}
	return t, nil
}

func (t *Threads) GetAllThreadsByChannelID(c *gin.Context, db *gorm.DB, userId, channelID string) ([]Threads, postgresql.PaginationResponse, error) {
	var (
		threads            []Threads
		ErrNotFound        = errors.New("threads not found")
		paginationResponse postgresql.PaginationResponse
	)

	var userChannels UserChannels
	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userId)
	if !exist {
		return nil, postgresql.PaginationResponse{}, errors.New("user not in channel")
	}

	pagination := postgresql.GetPagination(c)

	query := db.Model(&Threads{}).
		Select("threads.id, threads.channels_id, threads.full_name, threads.email, threads.event_name, threads.username, threads.content, threads.type, threads.avatar_url, threads.action_type, threads.created_at, threads.status, COUNT(messages) as message_count, MAX(messages.created_at) as last_reply").
		Joins("LEFT JOIN messages ON messages.thread_id = threads.id").
		Where("threads.channels_id = ?", channelID).
		Group("threads.id").
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Select("DISTINCT ON (messages.thread_id, messages.user_id) messages.*, profiles.avatar_url").
				Joins("LEFT JOIN profiles ON profiles.userid = messages.user_id").
				Order("messages.thread_id, messages.user_id, messages.created_at DESC").
				Limit(5)
		})

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"threads.created_at",
		"desc",
		pagination,
		&threads,
		nil,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paginationResponse, ErrNotFound
		}
		return nil, paginationResponse, errors.New("failed to fetch record")
	}

	return threads, paginationResponse, nil
}

func (t *Threads) GetThreadsByChannelID(c *gin.Context, db *gorm.DB, userId, channelID string) ([]Threads, postgresql.PaginationResponse, error) {
	var (
		threads            []Threads
		ErrNotFound        = errors.New("threads not found")
		paginationResponse postgresql.PaginationResponse
	)

	var userChannels UserChannels
	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userId)
	if !exist {
		return nil, postgresql.PaginationResponse{}, errors.New("user not in channel")
	}

	pagination := postgresql.GetPagination(c)

	query := db.Model(&Threads{}).
		Select("threads.id, threads.channels_id, threads.full_name, threads.email, threads.event_name, threads.username, threads.content, threads.type, threads.avatar_url, threads.action_type, threads.created_at, threads.status, COUNT(messages) as message_count, MAX(messages.created_at) as last_reply").
		Joins("LEFT JOIN messages ON messages.thread_id = threads.id").
		Where("threads.type = ? OR threads.type IS NULL", "thread").
		Where("threads.channels_id = ?", channelID).
		Group("threads.id").
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Select("DISTINCT ON (messages.thread_id, messages.user_id) messages.*, profiles.avatar_url").
				Joins("LEFT JOIN profiles ON profiles.userid = messages.user_id").
				Order("messages.thread_id, messages.user_id, messages.created_at DESC").
				Limit(5)
		})

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"threads.created_at",
		"desc",
		pagination,
		&threads,
		nil,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paginationResponse, ErrNotFound
		}
		return nil, paginationResponse, errors.New("failed to fetch record")
	}

	return threads, paginationResponse, nil
}

func (t *Threads) GetUserThreadsByOrganization(c *gin.Context, db *gorm.DB, userId, organisationID string) ([]Threads, postgresql.PaginationResponse, error) {
	var (
		threads     []Threads
		ErrNotFound = errors.New("threads not found")
	)

	query := db.Model(&Threads{}).
		Select("threads.id, threads.channels_id, threads.event_name, threads.type, threads.content, threads.current_status, channels.name as channel_name, threads.username, threads.avatar_url, threads.action_type, threads.created_at, threads.status, COUNT(messages) as message_count, MAX(messages.created_at) as last_reply").
		Joins("LEFT JOIN messages ON messages.thread_id = threads.id").
		Joins("LEFT JOIN channels ON threads.channels_id = channels.id").
		Where("channels.organisation_id = ?", organisationID).
		Where("(threads.type <> ?  OR threads.type IS NULL)", "message").
		Group("threads.id, channels.name").
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Select("DISTINCT ON (messages.thread_id, messages.user_id) messages.*, profiles.avatar_url").
				Joins("LEFT JOIN profiles ON profiles.userid = messages.user_id").
				Order("messages.thread_id, messages.user_id, messages.created_at DESC").
				Limit(5)
		})

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&threads,
		nil,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paginationResponse, ErrNotFound
		}
		return nil, paginationResponse, err
	}

	return threads, paginationResponse, nil
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
