package integrations

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateIntegrationApp(req models.Integrations, db *gorm.DB) (models.Integrations, error) {
    intApp := models.Integrations{
        ID:             utility.GenerateUUID(),
        Name:           req.Name,
        ApiEndpointUrl: req.ApiEndpointUrl,
        AuthCredential: req.AuthCredential,
    }

    err := intApp.CreateIntegrationApp(db, req.Name)
    if err != nil {
        return models.Integrations{}, err
    }

    return intApp, nil
}

func GetAllIntegrationApp(c *gin.Context, db *gorm.DB) ([]models.Integrations, postgresql.PaginationResponse, error) {
	intApp := models.Integrations{}
	intApps, paginationResponse, err := intApp.GetAllIntegrationApp(db, c)

	if err != nil {
		return nil, paginationResponse, err
	}

	return intApps, paginationResponse, nil
}