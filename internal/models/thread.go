package models

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Threads struct {
	ID           string    `gorm:"type:uuid;primary_key" json:"thread_id"`
	ChannelsID   string    `gorm:"type:uuid;index" json:"channels_id"`
	EventName    string    `gorm:"type:varchar(200);index" json:"event_name"`
	Username     string    `gorm:"type:varchar(50);index" json:"username"`
	ActionType   string    `gorm:"type:varchar(200);index" json:"action_type"`
	CreatedAt    time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	Messages     []Message `gorm:"foreignKey:ThreadID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"messages"`
	ThreadStatus string    `gorm:"type:varchar(20);" json:"thread_status"`
	MessageCount int64     `gorm:"type:int;" json:"message_count"`
}

type Mentions struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	MessageID string    `gorm:"type:uuid;index" json:"message_id"`
	UserID    string    `gorm:"type:uuid;index" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type UpdateThreadStatus struct {
	ThreadStatus string `json:"status" validate:"required"`
}

func (t *Threads) CreateThread(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, t)
	if err != nil {
		return err
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

func (t *Threads) GetThreadById(db *gorm.DB, ID string) (*Threads, error) {

	err, nerr := postgresql.SelectOneFromDb(db, &t, "id = ?", ID)
	if nerr != nil {
		return t, err
	}
	return t, nil
}

func (t *Threads) GetThreadByIds(db *gorm.DB, threadID, userID string) (*Threads, error) {

	err, nerr := postgresql.SelectOneFromDb(db, &t, "id = ? AND user_id = ?", threadID, userID)
	if nerr != nil {
		return t, err
	}
	return t, nil
}

func (t *Threads) GetThreadsByChannelID(c *gin.Context, db *gorm.DB, userId, channelID string) ([]Threads, postgresql.PaginationResponse, error) {
	var (
		threads     []Threads
		ErrNotFound = errors.New("threads not found")
	)

	var userChannels UserChannels
	exist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userId)
	if !exist {
		return nil, postgresql.PaginationResponse{}, errors.New("user not in channel")
	}

	pagination := postgresql.GetPagination(c)
	query := db.Model(&Threads{}).
		Select("threads.id, threads.channels_id, threads.user_id, threads.event_name, threads.username, threads.action_type, threads.created_at, threads.thread_status, COUNT(messages.id) as message_count").
		Joins("LEFT JOIN messages ON messages.thread_id = threads.id").
		Where("threads.channels_id = ?", channelID).
		Group("threads.id")

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
		return nil, paginationResponse, err
	}

	return threads, paginationResponse, nil
}

func (t *Threads) GetUserThreadsByOrganization(c *gin.Context, db *gorm.DB, userId, organisationID string) ([]Threads, postgresql.PaginationResponse, error) {
	var (
		threads     []Threads
		ErrNotFound = errors.New("threads not found")
	)

	var userChannels []UserChannels
	err := db.Joins("JOIN channels ON channels.id = user_channels.channels_id").
		Where("user_channels.user_id = ? AND channels.organisation_id = ?", userId, organisationID).
		Find(&userChannels).Error
	if err != nil {
		return nil, postgresql.PaginationResponse{}, errors.New("failed to retrieve user channels within the organization")
	}

	channelIDs := make([]string, len(userChannels))
	for i, uc := range userChannels {
		channelIDs[i] = uc.ChannelsID
	}

	pagination := postgresql.GetPagination(c)
	query := db.Where("channels_id IN (?)", channelIDs).
		Preload("Messages")

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

func (t *Threads) GetSingleThreadWithReplies(db *gorm.DB, threadID string) (*Threads, error) {
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
