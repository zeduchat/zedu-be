package agents

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func AddChannelAgentSettings(db *gorm.DB, ids map[string]string, req models.AddIntegrationSettingsRequest) error {

	is := models.ChannelIntegrationSettings{
		ID:             utility.GenerateUUID(),
		OrgID:          ids["org_id"],
		IntegrationID:  ids["agent_id"],
		ChannelID:      ids["channel_id"],
		FormFieldValue: req.FormFieldValue,
		FormFieldLabel: req.FormFieldLabel,
	}

	err := is.CreateChannelIntegrationSettings(db)
	if err != nil {
		return err
	}

	return nil
}

func GetChannelAgentSettings(db *gorm.DB, ids map[string]string) ([]models.ChannelIntegrationSettings, error) {
	var (
		organisation models.Organisation
		agent        models.Integrations
		channel      models.Channels
		setting      models.ChannelIntegrationSettings
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return []models.ChannelIntegrationSettings{}, fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &agent, "id = ?", ids["agent_id"])
	if !exists {
		return []models.ChannelIntegrationSettings{}, fmt.Errorf("agent does not exist")
	}

	exists = postgresql.CheckExists(db, &channel, "id = ?", ids["channel_id"])

	settings, err := setting.GetChannelIntegrationSetting(db, ids)
	if err != nil {
		return settings, err
	}
	return settings, nil
}

func UpdateChannelAgentSettings(db *gorm.DB, ids map[string]string, req models.UpdateIntegrationSettingsRequest) error {
	var (
		setting      models.ChannelIntegrationSettings
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

	err := setting.UpdateChannelIntegrationSetting(db, ids, req)
	if err != nil {
		return err
	}

	return nil
}

func DeleteChannelAgentSettings(db *gorm.DB, ids map[string]string) error {
	var (
		setting      models.ChannelIntegrationSettings
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

	exists = postgresql.CheckExists(db, &setting, "org_id = ? AND integration_id = ? AND channel_id = ? AND id = ?", ids["org_id"], ids["agent_id"], ids["channel_id"], ids["setting_id"])
	if !exists {
		return fmt.Errorf("channel agent setting does not exist")
	}

	err := setting.DeleteChannelIntegrationSetting(db, ids)
	if err != nil {
		return err
	}

	return nil
}
