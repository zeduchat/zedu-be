package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type DmFavourite struct {
	ID        string         `gorm:"type:uuid;primary_key" json:"id"`
	UserId    string         `gorm:"type:uuid;not null;uniqueIndex:idx_dm_fav_user_channel" json:"user_id"`
	ChannelId string         `gorm:"type:uuid;not null;uniqueIndex:idx_dm_fav_user_channel" json:"channel_id"`
	OrgId     string         `gorm:"type:uuid;not null" json:"org_id"`
	CreatedAt time.Time      `gorm:"type:timestamp;default:current_timestamp" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type DmFavouriteRequest struct {
	UserId    string `json:"user_id"`
	ChannelId string `json:"channel_id"`
	OrgId     string `json:"org_id"`
}

func (f *DmFavourite) AddToFavourites(db *gorm.DB, req DmFavouriteRequest) error {
	var existing DmFavourite
	exists := postgresql.CheckExists(db, &existing, "user_id = ? AND channel_id = ? AND org_id = ?", req.UserId, req.ChannelId, req.OrgId)
	if exists {
		*f = existing
		return nil
	}

	f.UserId = req.UserId
	f.ChannelId = req.ChannelId
	f.OrgId = req.OrgId

	return postgresql.CreateOneRecord(db, f)
}

func (f *DmFavourite) RemoveFromFavourites(db *gorm.DB, req DmFavouriteRequest) error {
	return db.Where("user_id = ? AND channel_id = ? AND org_id = ?", req.UserId, req.ChannelId, req.OrgId).Delete(&DmFavourite{}).Error
}
func IsFavourite(db *gorm.DB, userId, channelId string) bool {
	var fav DmFavourite
	return postgresql.CheckExists(db, &fav, "user_id = ? AND channel_id = ?", userId, channelId)
}

func GetUserFavouriteChannelIds(db *gorm.DB, userId, orgId string) ([]string, error) {
	var favourites []DmFavourite
	var channelIds []string

	err := db.Where("user_id = ? AND org_id = ? AND deleted_at IS NULL", userId, orgId).Find(&favourites).Error
	if err != nil {
		return channelIds, err
	}

	for _, fav := range favourites {
		channelIds = append(channelIds, fav.ChannelId)
	}

	return channelIds, nil
}
