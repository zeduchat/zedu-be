package models

import (
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type ChannelIntegrationSettings struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID          string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID  string    `gorm:"type:uuid;" json:"integration_id"`
	ChannelID      string    `gorm:"type:uuid;" json:"channel_id"`
	FormFieldValue string    `gorm:"column:form_field_value; type:varchar(255);" json:"form_field_value"`
	FormFieldLabel string    `gorm:"column:form_field_label; type:varchar(255);" json:"form_field_label"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type AddChannelIntegrationSettingsRequest struct {
	FormFieldValue string `json:"form_field_value" binding:"required"`
	FormFieldLabel string `json:"form_field_label" binding:"required"`
}

type UpdateChannelIntegrationSettingsRequest struct {
	FormFieldValue string `json:"form_field_value"`
	FormFieldLabel string `json:"form_field_label"`
}

func (is *ChannelIntegrationSettings) CreateChannelIntegrationSettings(db *gorm.DB) error {
	var (
		channel                 Channels
		organisationIntegration OrganisationIntegrations
	)

	exists := postgresql.CheckExists(db, &channel, "id = ? AND organisation_id = ?", is.ChannelID, is.OrgID)
	if !exists {
		return fmt.Errorf("channel does not exist in organisation")
	}

	exists = postgresql.CheckExists(db, &organisationIntegration, "org_id = ? AND integration_id = ?", is.OrgID, is.IntegrationID)
	if !exists {
		return fmt.Errorf("integration is not enabled for organisation")
	}

	err := postgresql.CreateOneRecord(db, &is)
	if err != nil {
		return fmt.Errorf("failed to create channel integration settings: %v", err)
	}
	return nil
}

func (is *ChannelIntegrationSettings) GetChannelIntegrationSetting(db *gorm.DB, ids map[string]string) ([]ChannelIntegrationSettings, error) {
	var intSettings []ChannelIntegrationSettings

	err := postgresql.SelectAllFromDb(db, "", &intSettings, "org_id = ? AND integration_id = ? AND channel_id = ?", ids["org_id"], ids["agent_id"], ids["channel_id"])
	if err != nil {
		return nil, fmt.Errorf("failed to get channel integration settings: %v", err)
	}
	return intSettings, nil
}

func (is *ChannelIntegrationSettings) UpdateChannelIntegrationSetting(db *gorm.DB, ids map[string]string, req UpdateIntegrationSettingsRequest) error {
	var (
		channel                 Channels
		organisationIntegration OrganisationIntegrations
	)

	exists := postgresql.CheckExists(db, &channel, "id = ? AND organisation_id = ?", ids["channel_id"], ids["org_id"])
	if !exists {
		return fmt.Errorf("channel does not exist in organisation")
	}

	exists = postgresql.CheckExists(db, &organisationIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return fmt.Errorf("integration is not enabled for organisation")
	}

	exists = postgresql.CheckExists(db, &is, "org_id = ? AND integration_id = ? AND channel_id = ? AND id = ?", ids["org_id"], ids["agent_id"], ids["channel_id"], ids["setting_id"])
	if !exists {
		return fmt.Errorf("channel integration setting does not exist")
	}

	record, err := postgresql.UpdateFields(db, &is, req, "id = ?", is.ID)
	if err != nil {
		return fmt.Errorf("failed to update channel integration settings: %v", err)
	}
	if record.RowsAffected == 0 {
		return fmt.Errorf("no record was updated")
	}
	return nil
}

func (is *ChannelIntegrationSettings) DeleteChannelIntegrationSetting(db *gorm.DB, ids map[string]string) error {

	exists := postgresql.CheckExists(db, &is, "org_id = ? AND integration_id = ? AND channel_id = ?", ids["org_id"], ids["agent_id"], ids["channel_id"])
	if !exists {
		return fmt.Errorf("channel integration setting does not exist")
	}

	err := postgresql.DeleteRecordFromDb(db, &is)
	if err != nil {
		return fmt.Errorf("failed to delete channel integration setting: %v", err)
	}
	return nil
}
