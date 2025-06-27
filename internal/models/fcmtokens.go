package models

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type FcmTokens struct {
	ID         string    `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	UserId     string    `gorm:"column:user_id; type:uuid; not null" json:"user_id"`
	FcmToken   string    `gorm:"column:fcm_token; type:text;" json:"fcm_token"`
	IsLive     bool      `gorm:"column:is_live; type:bool; default:false; not null" json:"is_live"`
	Type       string    `gorm:"column:type;default:fcm;type:text" json:"-"`
	WnsToken   string    `gorm:"column:wns_token;type:text"  json:"wns_token" validate:"required"`
	ChannelUri string    `gorm:"column:channel_uri;type:text" json:"channel_uri" validate:"required"`
	CreatedAt  time.Time `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}

type CreateFcmTokenRequest struct {
	UserId   string `json:"user_id"`
	FcmToken string `json:"fcm_token" validate:"required"`
}

type CreateWnsTokenRequest struct {
	UserId     string `json:"user_id"`
	WnsToken   string `json:"wns_token"`
	ChannelUri string `json:"channel_uri" validate:"required"`
}

func (ft *FcmTokens) CreateFcmToken(db *gorm.DB) error {

	exists := postgresql.CheckExists(db, &FcmTokens{}, "user_id = ?", ft.UserId)

	if !exists {
		err := postgresql.CreateOneRecord(db, &ft)
		if err != nil {
			return err
		}

		return nil
	}

	req_fields := make(map[string]any)
	req_fields["fcm_token"] = ft.FcmToken

	result, err := postgresql.UpdateFields(db, &FcmTokens{}, req_fields, "user_id = ?", ft.UserId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update user fcmtoken")
	}

	return nil
}

func (ft *FcmTokens) CreateWnsConfig(db *gorm.DB) error {

	exists := postgresql.CheckExists(db, &FcmTokens{}, "user_id = ?", ft.UserId)

	if !exists {
		err := postgresql.CreateOneRecord(db, &ft)
		if err != nil {
			return err
		}

		return nil
	}

	req_fields := make(map[string]any)
	req_fields["wns_token"] = ft.WnsToken
	req_fields["channel_uri"] = ft.ChannelUri

	result, err := postgresql.UpdateFields(db, &FcmTokens{}, req_fields, "user_id = ?", ft.UserId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update user fcmtoken")
	}

	return nil
}

func (ft *FcmTokens) GetFcmTokenByUserId(db *gorm.DB) (bool, error) {

	exists := postgresql.CheckExists(db, ft, "user_id = ?", ft.UserId)

	return exists, nil
}

func (ft *FcmTokens) GetFcmTokenByUserIds(db *gorm.DB, user_ids []string) ([]FcmTokens, error) {

	var resp []FcmTokens

	query := db.Where("user_id IN ?", user_ids)

	if err := query.Find(&resp).Error; err != nil {
		return resp, err
	}

	return resp, nil
}
