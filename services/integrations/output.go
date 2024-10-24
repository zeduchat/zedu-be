package integrations

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func GetActiveOutputIntegrations(db *gorm.DB, orgID string) ([]models.OutputIntegrationsResponse, error) {
	var (
		outputIntegrations []models.OutputIntegrationsResponse
		organisation       models.Organisation
	)

	exists := postgresql.CheckExists(db, &organisation, "id = ?", orgID)
	if !exists {
		return outputIntegrations, fmt.Errorf("organisation does not exist")
	}

	baseURL := "https://system-integration.telex.im/"
	err := db.Table("integrations").
		Select(fmt.Sprintf("integrations.id, integrations.name, CONCAT('%s', Lower(integrations.name), '/channels') AS channels_url", baseURL)).
		Joins("LEFT JOIN organisation_integrations ON organisation_integrations.integration_id = integrations.id").
		Where("organisation_integrations.org_id = ? AND organisation_integrations.is_active = TRUE AND integrations.integration_type = 'o'", orgID).
		Scan(&outputIntegrations).Error

	if err != nil {
		return nil, err
	}

	return outputIntegrations, nil
}
