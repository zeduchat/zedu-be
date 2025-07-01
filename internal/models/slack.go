package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type OAuth struct {
	OauthCode      string `json:"oauth_code,omitempty" validate:"required"`
	OrganisationID string `json:"organisation_id"`
}

type JSONB map[string]any

type SlackTelex struct {
	ID               string    `gorm:"type:uuid;primary_key" json:"id"`
	UserID           string    `gorm:"type:uuid;not null" json:"user_id"`
	OrganisationID   string    `gorm:"type:uuid;not null" json:"organisation_id"`
	IntegrationID    string    `gorm:"type:uuid" json:"integration_id"`
	AccessToken      string    `gorm:"type:text" json:"access_token,omitempty"`
	TeamID           string    `gorm:"type:text" json:"team_id,omitempty"`
	TeamName         string    `gorm:"type:text" json:"team_name,omitempty"`
	Channel          string    `gorm:"type:text" json:"channel,omitempty"`
	ChannelID        string    `gorm:"type:text" json:"channel_id,omitempty"`
	ConfigurationURL string    `gorm:"type:text" json:"configuration_url,omitempty"`
	URL              string    `gorm:"type:text" json:"url,omitempty"`
	AppManifest      JSONB     `gorm:"type:jsonb;serializer:json" json:"app_manifest,omitempty"`
	CreatedAt        time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type SendSlackRequest struct {
	OrgName        string `json:"org_name"`
	AuthorName     string `json:"author_name"`
	TitleEvent     string `json:"title_event"`
	TitleAction    string `json:"title_action"`
	TitleLink      string `json:"title_link"`
	PretextChannel string `json:"pretext_channel"`
	StatusValue    string `json:"status_value"`
	WebhookUrl     string `json:"webhook_url"`
	Color          string `json:"color"`
}

type SlackToken struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	AccessToken    string    `gorm:"type:text" json:"access_token"`
	RefreshToken   string    `gorm:"type:text" json:"refresh_token"`
	ExpiryTime     time.Time `gorm:"column:expiry_time; not null" json:"expiry_time"`
	UserID         string    `gorm:"type:uuid" json:"user_id"`
	OrganisationID string    `gorm:"type:uuid" json:"organisation_id"`
}

func (s *SlackTelex) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &s)

	if err != nil {
		return err
	}

	return nil
}

func (s *SlackTelex) GetSlackAccessToken(db *gorm.DB, userId string, orgId string) error {
	err, _ := postgresql.SelectOneFromDb(db, &s, "organisation_id = ?", orgId)

	if err != nil {
		return err
	}

	return nil
}

func (s *SlackTelex) GetSlackWebhookUrl(db *gorm.DB, orgId string) error {
	err, _ := postgresql.SelectOneFromDb(db, &s, "organisation_id = ?", orgId)

	if err != nil {
		return err
	}

	return nil
}

func (s *SlackToken) Create(db *gorm.DB, refresh_token string) error {
	s.RefreshToken = refresh_token
	s.ExpiryTime = time.Now().Add(time.Hour * 11)

	err := postgresql.CreateOneRecord(db, &s)

	if err != nil {
		return err
	}

	return nil
}

func (s *SlackToken) IsEmpty(db *gorm.DB) bool {
	err := db.First(&s)

	return err.Error != nil
}

func (s *SlackToken) GetSlackToken(db *gorm.DB) error {
	err := db.First(&s)

	if err.Error != nil {
		return err.Error
	}

	return nil
}

func (s *SlackToken) GetToken(db *gorm.DB, userId, orgId string) (SlackToken, error) {
	err := db.First(&s, "user_id = ? AND organisation_id = ?", userId, orgId)

	if err.Error != nil {
		return SlackToken{}, err.Error
	}

	return *s, nil
}

func (s *SlackToken) UpdateToken(db *gorm.DB, userId, orgId string, accessToken, refreshToken string) error {
	updates := map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expiry_time":   time.Now().Add(time.Hour * 11),
	}

	err := db.Model(&s).Where("user_id = ? AND organisation_id = ?", userId, orgId).Updates(updates)

	if err.Error != nil {
		return err.Error
	}

	return nil
}
