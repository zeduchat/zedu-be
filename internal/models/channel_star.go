package models

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type ChannelStar struct {
	ID           string         `gorm:"type:uuid;primary_key" json:"id"`
	UserId       string         `gorm:"type:uuid;not null;uniqueIndex:idx_channel_star_user_channel,where:deleted_at IS NULL" json:"user_id"`
	ChannelId    string         `gorm:"type:uuid;not null;uniqueIndex:idx_channel_star_user_channel,where:deleted_at IS NULL" json:"channel_id"`
	OrgId        string         `gorm:"type:uuid;not null" json:"org_id"`
	CreatedAt    time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	User         User           `gorm:"foreignKey:UserId;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Channel      Channels       `gorm:"foreignKey:ChannelId;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Organisation Organisation   `gorm:"foreignKey:OrgId;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (ChannelStar) TableName() string {
	return "channel_stars"
}

type ChannelStarRequest struct {
	UserId    string `json:"user_id"`
	ChannelId string `json:"channel_id"`
	OrgId     string `json:"org_id"`
}

func (s *ChannelStar) AddToStars(db *gorm.DB, req ChannelStarRequest) error {
	var existing ChannelStar
	exists := postgresql.CheckExists(db, &existing, "user_id = ? AND channel_id = ? AND org_id = ?", req.UserId, req.ChannelId, req.OrgId)
	if exists {
		*s = existing
		return nil
	}

	s.UserId = req.UserId
	s.ChannelId = req.ChannelId
	s.OrgId = req.OrgId

	if err := postgresql.CreateOneRecord(db, s); err != nil {
		// A concurrent request may have starred it first
		if s.IsStarChannel(db, req) {
			return nil
		}
		return err
	}
	return nil
}

func (s *ChannelStar) RemoveFromStars(db *gorm.DB, req ChannelStarRequest) error {
	return postgresql.HardDeleteSpecificRecord(db, &ChannelStar{}, "user_id = ? AND channel_id = ? AND org_id = ?", req.UserId, req.ChannelId, req.OrgId)
}

func (s *ChannelStar) IsStarChannel(db *gorm.DB, req ChannelStarRequest) bool {
	return postgresql.CheckExists(db, &ChannelStar{}, "user_id = ? AND channel_id = ? AND org_id = ?", req.UserId, req.ChannelId, req.OrgId)
}

func IsChannelStarred(db *gorm.DB, userId, channelId string) (bool, error) {
	var star ChannelStar
	err, _ := postgresql.SelectOneFromDb(db, &star, "user_id = ? AND channel_id = ?", userId, channelId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func GetUserStarredChannelIds(db *gorm.DB, userId, orgId string) ([]string, error) {
	var stars []ChannelStar
	var channelIds []string

	err := db.Where("user_id = ? AND org_id = ? AND deleted_at IS NULL", userId, orgId).Find(&stars).Error
	if err != nil {
		return channelIds, err
	}

	for _, star := range stars {
		channelIds = append(channelIds, star.ChannelId)
	}

	return channelIds, nil
}
