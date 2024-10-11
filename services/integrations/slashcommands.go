package integrations

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func AddIntegrationsSlashCommand(db *gorm.DB, ids map[string]string, req models.AddSlashCommandRequest) (models.SlashCommand, error) {

	slashCommand := models.SlashCommand{
		ID:            utility.GenerateUUID(),
		OrgID:         ids["org_id"],
		IntegrationID: ids["integration_id"],
		Command:       req.Command,
		ProcessingURL: req.ProcessingURL,
		Description:   req.Description,
	}

	response, err := slashCommand.CreateSlashCommand(db)
	if err != nil {
		return response, err
	}

	return response, nil
}

func GetIntegrationSlashCommands(db *gorm.DB, ids map[string]string) ([]models.SlashCommand, error) {
	var (
		slashCommand models.SlashCommand
	)

	response, err := slashCommand.GetIntegrationSlashCommands(db, ids)
	if err != nil {
		return response, err
	}
	return response, nil
}

func GetAllOrgSlashCommands(db *gorm.DB, orgID string) ([]models.SlashCommand, error) {
	var (
		slashCommand models.SlashCommand
	)

	response, err := slashCommand.GetAllOrgSlashCommands(db, orgID)
	if err != nil {
		return response, err
	}
	return response, nil
}

func UpdateIntegrationSlashCommand(db *gorm.DB, ids map[string]string, req models.UpdateSlashCommandRequest) (models.SlashCommand, error) {
	var (
		slashCommand models.SlashCommand
	)

	response, err := slashCommand.UpdateSlashCommand(db, ids, req)
	if err != nil {
		return response, err
	}
	return response, err
}

func DeleteIntegrationSlashCommand(db *gorm.DB, ids map[string]string) error {
	var (
		slashCommand models.SlashCommand
	)

	err := slashCommand.DeleteSlashCommand(db, ids)
	if err != nil {
		return err
	}
	return nil
}
