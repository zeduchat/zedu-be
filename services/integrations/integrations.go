package integrations

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

func GetAllIntegrationApp(c *gin.Context, org_id string, db *gorm.DB) ([]models.Integrations, error) {
	integration := models.Integrations{}
	integrations, err := integration.GetAllIntegrationApp(db, org_id, c)

	if err != nil {
		return nil, err
	}

	return integrations, nil
}

func UpdateIntegrationApp(req models.UpdateIntegration, ids map[string]string, db *gorm.DB) (models.Integrations, error) {
	var integration models.Integrations

	updatedIntegration, err := integration.UpdateIntegration(db, ids, req)
	if err != nil {
		return models.Integrations{}, err
	}

	return updatedIntegration, nil
}

func DeleteIntegrationApp(ids map[string]string, db *gorm.DB) error {
	var integration models.Integrations

	err := integration.DeleteIntegration(db, ids)
	if err != nil {
		return err
	}

	return nil
}

func ChangeIntegrationStatus(ids map[string]string, req models.ChangeIntegrationStatus, db *gorm.DB) error {
	var integration models.OrganisationIntegrations


	err := integration.ChangeStatus(db, req , ids)
	if err != nil {
		return err
	}

	return nil
}

func UpdateJSONSchema(ids map[string]string, req models.UpdateJSONSchemaRequest, db *gorm.DB) error {
	var orgIntegration models.OrganisationIntegrations

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["integration_id"])
	if !exists {
		return errors.New("organisation does not have that integration")
	}

	err := orgIntegration.UpdateJSONSchema(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}

func GetOrganisationChannelIntegrations(db *gorm.DB, channel_id, org_id string, c *gin.Context) ([]models.Integrations, postgresql.PaginationResponse, error) {
	var ocIntegrations models.OrganisationChannelsIntegrations

	integrations, paginationResponse, err := ocIntegrations.GetOrganisationChannelIntegrations(db, channel_id, org_id, c)
	if err != nil {
		return nil, paginationResponse, err
	}

	return integrations, paginationResponse, nil
}

func ActivateChannelIntegration(ids map[string]string, req models.ActivateChannelIntegration, db *gorm.DB) error {
	var (
		ocIntegrations  models.OrganisationChannelsIntegrations
		orgIntegrations models.OrganisationIntegrations
		channels        models.Channels
	)

	exists := postgresql.CheckExists(db, &orgIntegrations, "org_id = ? AND integration_id = ?", ids["organisation_id"], ids["integration_id"])
	if !exists {
		return errors.New("organisation does not have that integration")
	}

	exists = postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", ids["channel_id"], ids["organisation_id"])
	if !exists {
		return errors.New("organisation does not have that channel")
	}

	err := ocIntegrations.ActivateChannelIntegration(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}
