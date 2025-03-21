package agents

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func AddAgentSettings(db *gorm.DB, ids map[string]string, req models.AddIntegrationSettingsRequest) error {

	is := models.IntegrationSettings{
		ID:             utility.GenerateUUID(),
		OrgID:          ids["org_id"],
		IntegrationID:  ids["agent_id"],
		FormFieldValue: req.FormFieldValue,
		FormFieldLabel: req.FormFieldLabel,
	}

	err := is.CreateIntegrationSettings(db)
	if err != nil {
		return err
	}

	return nil
}

func GetAgentSetting(db *gorm.DB, ids map[string]string) ([]models.IntegrationSettings, error) {
	var (
		organisation models.Organisation
		agent        models.Integrations
		setting      models.IntegrationSettings
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("agent does not exist")
	}

	settings, err := setting.GetIntegrationSetting(db, ids)
	if err != nil {
		return settings, err
	}
	return settings, nil
}

func GetCustomAgentApiKey(db *gorm.DB, ids map[string]string) (string, int, error) {
	var (
		organisation models.Organisation
		agent        models.Integrations
		setting      models.CustomIntegrationsSetting
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return "", http.StatusNotFound, fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	if !exists {
		return "", http.StatusNotFound, fmt.Errorf("agent does not exist")
	}

	api_key, code, err := setting.GetIntegrationApiKey(db, ids)
	if err != nil {
		return "", code, err
	}

	return api_key, code, nil
}

func UpdateAgentSettings(db *gorm.DB, ids map[string]string, req models.UpdateIntegrationSettingsRequest) error {
	var (
		setting      models.IntegrationSettings
		organisation models.Organisation
		agent        models.Integrations
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	if !exists {
		return fmt.Errorf("agent does not exist")
	}

	err := setting.UpdateIntegrationSetting(db, ids, req)
	if err != nil {
		return err
	}

	return nil
}

func GetOrgAgentSettings(db *gorm.DB, ids map[string]string) ([]models.IntegrationSettings, error) {
	var (
		organisation models.Organisation
		agent        models.Integrations
		setting      models.IntegrationSettings
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	if !exists {
		return []models.IntegrationSettings{}, fmt.Errorf("agent does not exist")
	}

	settings, err := setting.GetIntegrationSetting(db, ids)
	if err != nil {
		return settings, err
	}
	return settings, nil
}

func GetAgentSettingsAllOrgs(db *gorm.DB, agent_id string) ([]models.IntegrationSettings, error) {
	var setting models.IntegrationSettings

	settings, err := setting.GetIntegrationSettingsAllOrgs(db, agent_id)
	if err != nil {
		return settings, err
	}
	return settings, nil
}
