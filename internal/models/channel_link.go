package models

import (
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type ChannelLink struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID      string    `gorm:"type:uuid;column:channel_id;not null;index:idx_channel_links_channel_created" json:"channel_id"`
	ThreadID       string    `gorm:"type:uuid;column:thread_id;not null;index" json:"thread_id"`
	MessageID      string    `gorm:"type:uuid;column:message_id;not null;uniqueIndex:idx_channel_links_message_url" json:"message_id"`
	UserID         string    `gorm:"type:uuid;column:user_id;not null;index" json:"user_id"`
	OrganisationID string    `gorm:"type:uuid;column:organisation_id;not null;index" json:"organisation_id"`
	URL            string    `gorm:"type:varchar(2048);column:url;not null;uniqueIndex:idx_channel_links_message_url" json:"url"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime;index:idx_channel_links_channel_created,sort:desc" json:"created_at"`
}

type ChannelLinkResponse struct {
	URL       string    `json:"url"`
	MessageID string    `json:"message_id"`
	ThreadID  string    `json:"thread_id"`
	ChannelID string    `json:"channel_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

func (cl *ChannelLink) TableName() string {
	return "channel_links"
}

func (cl *ChannelLink) BeforeCreate(tx *gorm.DB) error {
	if cl.ID == "" {
		cl.ID = utility.GenerateUUID()
	}
	return nil
}

func ReplaceMessageLinks(db *gorm.DB, messageID string, links []ChannelLink) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("message_id = ?", messageID).Delete(&ChannelLink{}).Error; err != nil {
			return err
		}
		if len(links) == 0 {
			return nil
		}
		return tx.Create(&links).Error
	})
}

func DeleteChannelLinksByMessage(db *gorm.DB, messageID string) error {
	return db.Where("message_id = ?", messageID).Delete(&ChannelLink{}).Error
}

func DeleteChannelLinksByThread(db *gorm.DB, threadID string) error {
	return db.Where("thread_id = ?", threadID).Delete(&ChannelLink{}).Error
}

func DeleteChannelLinksByChannel(db *gorm.DB, channelID string) error {
	return db.Where("channel_id = ?", channelID).Delete(&ChannelLink{}).Error
}

func SelectChannelLinks(db *gorm.DB, channelID string, page, limit int) ([]ChannelLink, postgresql.PaginationResponse, error) {
	var links []ChannelLink
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	var totalCount int64
	if err := db.Model(&ChannelLink{}).
		Where("channel_id = ?", channelID).
		Count(&totalCount).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	if err := db.Where("channel_id = ?", channelID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&links).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	pagination := postgresql.PaginationResponse{
		CurrentPage:     page,
		PageCount:       limit,
		TotalPagesCount: int(math.Ceil(float64(totalCount) / float64(limit))),
		TotalItems:      totalCount,
	}

	return links, pagination, nil
}
