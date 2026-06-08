package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type FcmTokens struct {
	ID          string      `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	UserId      string      `gorm:"column:user_id; type:uuid; not null" json:"user_id"`
	FcmToken    string      `gorm:"column:fcm_token; type:text;" json:"fcm_token"`
	VoIPToken   string      `gorm:"column:voip_token; type:text;" json:"voip_token"`
	IsLive      bool        `gorm:"column:is_live; type:bool; default:false; not null" json:"is_live"`
	Type        string      `gorm:"column:type;default:fcm;type:text" json:"-"`
	WnsToken    string      `gorm:"column:wns_token;type:text"  json:"wns_token" validate:"required"`
	ChannelUri  string      `gorm:"column:channel_uri;type:text" json:"channel_uri" validate:"required"`
	SubsPayload SubsPayload `gorm:"type:jsonb" json:"web_sub_payload"`
	CreatedAt   time.Time   `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time   `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}

type VoIPTokenRequest struct {
	VoIPToken string `json:"voip_token" validate:"required"`
}

type CreateFcmTokenRequest struct {
	UserId   string `json:"user_id"`
	FcmToken string `json:"fcm_token" validate:"required"`
}

type CreateWebPushTokenRequest struct {
	UserId   string `json:"user_id"`
	Endpoint string `json:"endpoint" validate:"required"`
	P256dh   string `json:"p256dh" validate:"required"`
	Auth     string `json:"auth" validate:"required"`
}

type CreateWnsTokenRequest struct {
	UserId     string `json:"user_id"`
	WnsToken   string `json:"wns_token"`
	ChannelUri string `json:"channel_uri" validate:"required"`
}

type SubsPayload struct {
	Endpoint string `json:"endpoint" validate:"required"`
	Keys     Keys   `json:"keys" validate:"required"`
}
type Keys struct {
	P256dh string `json:"p256dh" validate:"required"`
	Auth   string `json:"auth" validate:"required"`
}

func (p *SubsPayload) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal webpushpayload: value is not []byte")
	}

	return json.Unmarshal(bytes, p)
}

func (p SubsPayload) Value() (driver.Value, error) {
	return json.Marshal(p)
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

func (ft *FcmTokens) GetFcmTokenByUserIds(db *gorm.DB, user_ids []string) (*[]FcmTokens, error) {

	var resp []FcmTokens

	query := db.Where("user_id IN ?", user_ids)

	if err := query.Find(&resp).Error; err != nil {
		return &resp, err
	}

	return &resp, nil
}

func (ft *FcmTokens) CreateWebPushConfig(db *gorm.DB) error {

	exists := postgresql.CheckExists(db, &FcmTokens{}, "user_id = ?", ft.UserId)

	if !exists {
		err := postgresql.CreateOneRecord(db, &ft)
		if err != nil {
			return err
		}

		return nil
	}

	req_fields := make(map[string]any)
	req_fields["subs_payload"] = ft.SubsPayload

	result, err := postgresql.UpdateFields(db, &FcmTokens{}, req_fields, "user_id = ?", ft.UserId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update user web push config")
	}

	return nil
}

func (ft *FcmTokens) GetValidWebPushTokenByUserId(db *gorm.DB) (bool, error) {
	// Validity period = 24h from created_at
	cutoff := time.Now().Add(-24 * time.Hour)

	exists := postgresql.CheckExists(
		db,
		ft,
		"user_id = ? AND updated_at >= ?",
		ft.UserId,
		cutoff,
	)

	return exists, nil
}

func (ft *FcmTokens) GetValidWebPushTokenByUserIds(db *gorm.DB, user_ids []string) (*[]FcmTokens, error) {

	var resp []FcmTokens
	cutoff := time.Now().Add(-24 * time.Hour)

	query := db.Where("user_id = ? AND updated_at >= ?", user_ids, cutoff)

	if err := query.Find(&resp).Error; err != nil {
		return &resp, err
	}

	return &resp, nil
}

func (ft *FcmTokens) CreateVoIPToken(db *gorm.DB) error {
	exists := postgresql.CheckExists(db, &FcmTokens{}, "user_id = ?", ft.UserId)

	if !exists {
		err := postgresql.CreateOneRecord(db, &ft)
		if err != nil {
			return err
		}
		return nil
	}

	req_fields := make(map[string]any)
	req_fields["voip_token"] = ft.VoIPToken

	result, err := postgresql.UpdateFields(db, &FcmTokens{}, req_fields, "user_id = ?", ft.UserId)
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update user voip token")
	}

	return nil
}

func (ft *FcmTokens) ClearVoIPToken(db *gorm.DB) error {
	req_fields := make(map[string]any)
	req_fields["voip_token"] = ""

	_, err := postgresql.UpdateFields(db, &FcmTokens{}, req_fields, "voip_token = ?", ft.VoIPToken)
	return err
}
