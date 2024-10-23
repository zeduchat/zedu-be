package integrations

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func GetActiveOutputIntegrations(db *gorm.DB, orgID string) ([]models.OutputIntegrationsResponse, error) {
	// var outputIntegrations []models.OutputIntegrationsResponse

	// baseURL := "https://system-integrations.telex.im/"

	// err := db.Table("integrations").
	// 	Select("integrations.id, integrations.name, CONCAT(?, integrations.name, '/channels') AS channels_url", baseURL).
	// 	Joins("JOIN organisation_integrations ON organisation_integrations.integration_id = integrations.id").
	// 	Where("organisation_integrations.org_id = ? AND organisation_integrations.is_active = TRUE AND integrations.is_active = TRUE AND integrations.integration_type = ?", orgID, "o").
	// 	Scan(&outputIntegrations).Error

	// if err != nil {
	// 	return nil, err
	// }
	var (
		outputIntegrations []models.OutputIntegrationsResponse
		organisation       models.Organisation
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", orgID)
	if !exists {
		return outputIntegrations, fmt.Errorf("organisation does not exist")
	}

	err := db.Table("integrations").
		Select("integrations.id, integrations.name").
		Joins("JOIN organisation_integrations ON organisation_integrations.integration_id = integrations.id").
		Where("organisation_integrations.org_id = ? AND organisation_integrations.is_active = TRUE AND integrations.is_active = TRUE AND integrations.integration_type = ?", orgID, "o").
		Scan(&outputIntegrations).Error
	if err != nil {
		return nil, err
	}

	fmt.Println("Output------------", outputIntegrations)

	baseURL := "https://system-integrations.telex.im/"
	for i := range outputIntegrations {
		outputIntegrations[i].ChannelsUrl = baseURL + outputIntegrations[i].Name + "/channels"
	}

	return outputIntegrations, nil
}
