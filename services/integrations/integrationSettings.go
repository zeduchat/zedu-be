package integrations

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func AddIntegrationSettings(db *gorm.DB, ids map[string]string ,req models.AddIntegrationSettingsRequest) error {

	is := models.IntegrationSettings{
		ID:             utility.GenerateUUID(),
		OrgID:          ids["org_id"],
		IntegrationID:  ids["integration_id"],
		FormFieldValue: req.FormFieldValue,
		FormFieldLabel: req.FormFieldLabel,
	}

	err := is.CreateIntegrationSettings(db)
	if err != nil {
		return err
	}

	return nil
}

func GetIntegrationSetting(db *gorm.DB, ids map[string]string) ([]models.IntegrationSettings, error) {
	var (
		organisation models.Organisation
		integration models.Integrations
		setting models.IntegrationSettings
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("integration does not exist")
	}

	settings, err := setting.GetIntegrationSetting(db, ids)
	if err != nil {
		return settings, err
	}
	return settings, nil
}

func UpdateIntegrationSettings(db *gorm.DB, ids map[string]string ,req models.UpdateIntegrationSettingsRequest) error {
	var (
		setting models.IntegrationSettings
		organisation models.Organisation
		integration models.Integrations
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	if !exists {
		return fmt.Errorf("integration does not exist")
	}

	err := setting.UpdateIntegrationSetting(db, ids ,req)
	if err != nil {
		return err
	}

	return nil
}

func GetOrgIntegrationSettings(db *gorm.DB, ids map[string]string) ([]models.IntegrationSettings, error) {
	var (
		organisation models.Organisation
		integration models.Integrations
		setting models.IntegrationSettings
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &integration, "id = ?", ids["integration_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("integration does not exist")
	}

	settings, err := setting.GetIntegrationSetting(db, ids)
	if err != nil {
		return settings, err
	}
	return settings, nil
}

func GetIntegrationSettingsAllOrgs(db *gorm.DB, integration_id string) ([]models.IntegrationSettings, error) {
	var setting models.IntegrationSettings

	settings, err := setting.GetIntegrationSettingsAllOrgs(db, integration_id)
	if err != nil {
		return settings, err
	}
	return settings, nil
}