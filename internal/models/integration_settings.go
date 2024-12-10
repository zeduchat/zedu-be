package models

import (
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type IntegrationSettings struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	OrgID          string    `gorm:"type:uuid;" json:"org_id"`
	IntegrationID  string    `gorm:"type:uuid;" json:"integration_id"`
	FormFieldValue string    `gorm:"column:form_field_value; type:varchar(255);" json:"form_field_value"`
	FormFieldLabel string    `gorm:"column:form_field_label; type:varchar(255);" json:"form_field_label"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type AddIntegrationSettingsRequest struct {
	FormFieldValue string `json:"form_field_value" binding:"required"`
	FormFieldLabel string `json:"form_field_label" binding:"required"`
}


type UpdateIntegrationSettingsRequest struct {
	FormFieldValue string `json:"form_field_value"`
	FormFieldLabel string `json:"form_field_label"`
}

func (is *IntegrationSettings) CreateIntegrationSettings(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &is)
	if err != nil {
		return fmt.Errorf("failed to create integration settings: %v", err)
	}
	return nil
}

func (us *IntegrationSettings) GetIntegrationSettingsAllOrgs(db *gorm.DB, integration_id string) ([]IntegrationSettings, error) {
	var intSettings []IntegrationSettings

	err := postgresql.SelectAllFromDb(db, "", &intSettings, "integration_id = ?", integration_id)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration settings: %v", err)
	}
	return intSettings, nil
}

func (is *IntegrationSettings) GetIntegrationSetting(db *gorm.DB, ids map[string]string) ([]IntegrationSettings,error) {
	var intSettings []IntegrationSettings

	err := postgresql.SelectAllFromDb(db, "", &intSettings, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if err != nil {
		return nil, fmt.Errorf("failed to get integration settings: %v", err)
	}
	return intSettings, nil
}

func (is *IntegrationSettings) UpdateIntegrationSetting(db *gorm.DB, ids map[string]string ,req UpdateIntegrationSettingsRequest) error {

	exists := postgresql.CheckExists(db, &is, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return fmt.Errorf("integration setting does not exist")
	}

	record, err := postgresql.UpdateFields(db, &is, req, "id = ?", is.ID)
	if err != nil {
		return fmt.Errorf("failed to update integration settings: %v", err)
	}
	if record.RowsAffected == 0 {
		return fmt.Errorf("no record was updated")
	}
	return nil
}
