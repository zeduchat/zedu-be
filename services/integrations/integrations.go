package integrations

import (
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
		LogoUrl:             req.LogoUrl,
		ApiEndpointUrl:      req.ApiEndpointUrl,
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

func GetAllIntegrationApp(c *gin.Context, db *gorm.DB) ([]models.Integrations, postgresql.PaginationResponse, error) {
	integration := models.Integrations{}
	integrations, paginationResponse, err := integration.GetAllIntegrationApp(db, c)

	if err != nil {
		return nil, paginationResponse, err
	}

	return integrations, paginationResponse, nil
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

func GetOrganisationChannelIntegrations(db *gorm.DB,channel_id, org_id string, c *gin.Context) ([]models.Integrations, postgresql.PaginationResponse ,error) {
	var ocIntegrations models.OrganisationChannelsIntegrations

	integrations,paginationResponse ,err := ocIntegrations.GetOrganisationChannelIntegrations(db, channel_id, org_id, c)	
	if err != nil {
		return nil, paginationResponse, err
	}

	return integrations, paginationResponse, nil
}

func SlackIntegrationApp(req models.Integrations, db *gorm.DB) (models.Integrations, error) {
	intApp := models.Integrations{
		ID:             utility.GenerateUUID(),
		Name:           req.Name,
		ApiEndpointUrl: req.ApiEndpointUrl,
		AuthCredential: req.AuthCredential,
	}

	doc, err := goquery.NewDocument(req.ApiEndpointUrl)
	if err != nil {
		return models.Integrations{}, err
	}

	ogImage, exists := doc.Find("meta[property='og:image']").Attr("content")

	if exists {
		intApp.LogoUrl = ogImage
	} else {
		log.Println("No og:image found in the provided URL")
	}

	err = intApp.CreateSlackIntegration(db, req.Name)
	if err != nil {
		return models.Integrations{}, err
	}

	return intApp, nil
}
