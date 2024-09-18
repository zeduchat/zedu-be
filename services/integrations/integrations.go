package integrations

import (
	"errors"
	"log"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateIntegrationApp(req models.Integrations, org_id string, db *gorm.DB) (models.Integrations, error) {
	integration := models.Integrations{
		ID:                  utility.GenerateUUID(),
		Name:                req.Name,
		JSONUrl:             req.JSONUrl,
		JSONSchema:          req.JSONSchema,
		IntegrationType:     req.IntegrationType,
		AuthCredential:      req.AuthCredential,
		IsSystemIntegration: false,
	}

	err := integration.CreateIntegration(db, req)
	if err != nil {
		return models.Integrations{}, err
	}

	orgIntegration := models.OrganisationIntegrations{
		ID:            utility.GenerateUUID(),
		OrgID:         org_id,
		IntegrationID: integration.ID,
		IsArchived:    false,
		IsActive:      true,
	}

	err = orgIntegration.CreateOrganisationIntegration(db)
	if err != nil {
		return models.Integrations{}, err
	}

	return integration, nil
}

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

func SetIntegrationAppStatus(ids map[string]string, status string, db *gorm.DB) error {
	var integration models.OrganisationIntegrations

	err := integration.SetIntegrationStatus(db, status, ids)
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

func SlackIntegrationApp(req models.Integrations, db *gorm.DB) (models.Integrations, error) {
	intApp := models.Integrations{
		ID:             utility.GenerateUUID(),
		Name:           req.Name,
		JSONUrl:        req.JSONUrl,
		AuthCredential: req.AuthCredential,
	}

	doc, err := goquery.NewDocument(req.JSONUrl)
	if err != nil {
		return models.Integrations{}, err
	}

	ogImage, exists := doc.Find("meta[property='og:image']").Attr("content")

	if exists {
		// intApp.LogoUrl = ogImage
		_ = ogImage
	} else {
		log.Println("No og:image found in the provided URL")
	}

	err = intApp.CreateSlackIntegration(db, req.Name)
	if err != nil {
		return models.Integrations{}, err
	}

	return intApp, nil
}
