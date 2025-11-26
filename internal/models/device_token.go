package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)

// UserDeviceToken represents a device token tied to a user and optionally a device identifier/platform.
type UserDeviceToken struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      string    `gorm:"type:uuid;not null;index" json:"user_id"`
	DeviceToken string    `gorm:"type:text;not null" json:"device_token"`
	Platform    string    `gorm:"type:text" json:"platform,omitempty"`
	DeviceID    string    `gorm:"type:text" json:"device_id,omitempty"`
	LastSeen    time.Time `gorm:"column:last_seen;not null" json:"last_seen"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (u *UserDeviceToken) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = utility.GenerateUUID()
	}
	if u.LastSeen.IsZero() {
		u.LastSeen = time.Now().UTC()
	}
	return nil
}

type RegisterDeviceTokenRequest struct {
	DeviceToken string `json:"device_token" validate:"required"`
	Platform    string `json:"platform,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
}

type RegisterDeviceTokenResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	DeviceToken string    `json:"device_token"`
	Platform    string    `json:"platform,omitempty"`
	DeviceID    string    `json:"device_id,omitempty"`
	LastSeen    time.Time `json:"last_seen"`
	UpdatedAt   time.Time `json:"updated_at"`
}
