package models

import (
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type OAuth struct {
	OauthCode      string `json:"oauth_code,omitempty" validate:"required"`
	OrganisationID string `json:"organisation_id" validate:"required"`
}

type SlackTelex struct {
	ID               string    `gorm:"type:uuid;primary_key" json:"id"`
	UserID           string    `gorm:"type:uuid;not null" json:"user_id"`
	OrganisationID   string    `gorm:"type:uuid;not null" json:"organisation_id"`
	AccessToken      string    `gorm:"type:text" json:"access_token,omitempty"`
	TeamID           string    `gorm:"type:text" json:"team_id,omitempty"`
	TeamName         string    `gorm:"type:text" json:"team_name,omitempty"`
	Channel          string    `gorm:"type:text" json:"channel,omitempty"`
	ChannelID        string    `gorm:"type:text" json:"channel_id,omitempty"`
	ConfigurationURL string    `gorm:"type:text" json:"configuration_url,omitempty"`
	URL              string    `gorm:"type:text" json:"url,omitempty"`
	CreatedAt        time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

func (s *SlackTelex) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &s)

	if err != nil {
		return err
	}

	return nil
}

func (s *SlackTelex) GetSlackAccessToken(db *gorm.DB, userId string, orgId string) error {
	err, _ := postgresql.SelectOneFromDb(db, &s, "user_id = ? AND organisation_id = ?", userId, orgId)

	if err != nil {
		return err
	}

	return nil
}
