package models

import (
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type TelexSlackChannelMapping struct {
	ID               string    `gorm:"type:uuid;primary_key" json:"id"`
	UserID           string    `gorm:"type:uuid;not null" json:"user_id"`
	OrganisationID   string    `gorm:"type:uuid;not null" json:"organisation_id"`
	TelexChannelName string    `gorm:"type:varchar(255);not null" json:"telex_channel_name"`
	SlackChannelName string    `gorm:"type:varchar(255);not null" json:"slack_channel_name"`
	CreatedAt        time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type TelexSlackChannelMappingReq struct {
	TelexChannelName string `json:"telex_channel_name" validate:"required"`
	SlackChannelName string `json:"slack_channel_name" validate:"required"`
}

func (s *TelexSlackChannelMapping) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &s)

	if err != nil {
		return err
	}

	return nil
}
