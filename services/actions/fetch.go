package actions

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)

func FetchAPIKeyCredentials(c *gin.Context) (models.IDS, error) {
	var idmodel models.IDS

	// agentID, ok := c.Get("agent_id")
	// if !ok {
	// 	return idmodel, errors.New("agent_id is required")
	// }

	organisationID, ok := c.Get("org_id")
	if !ok {
		return idmodel, errors.New("organisation_id is required")
	}

	ids := models.IDS{
		OrganisationID: organisationID.(string),
	}

	return ids, nil
}

func HasEnoughCredits(db *gorm.DB, userID string, model string, messages []external_models.TelexAIOpenRouterMessage) bool {
	// This function checks if the user has enough credits to make the request.
	// For now, using true as a placeholder.
	return true
}
