package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func FetchOrganisationBots(db *gorm.DB, logger *utility.Logger, org_id string, c *gin.Context, extReq request.ExternalRequest, redisClient *redis.Client) ([]models.AgentResp, postgresql.PaginationResponse, int, error) {
	var (
		orgInt  models.OrganisationIntegrations
		botResp []models.AgentResp = make([]models.AgentResp, 0)
	)

	resp, paginatedResponse, err, code := orgInt.GetCustomAgentApps(db, org_id, c)
	if err != nil {
		return []models.AgentResp{}, paginatedResponse, code, err
	}

	for _, org_agents := range resp {

		agent := models.AgentResp{

			ID:          org_agents.IntegrationID,
			Name:        org_agents.AppName,
			Title:       org_agents.Title,
			Tone:        org_agents.Tone,
			Visibility:  org_agents.Visibility,
			Avatar:      org_agents.AppLogo,
			Description: org_agents.AppDescription,
			IsActive:    org_agents.IsActive,
		}

		botResp = append(botResp, agent)

	}

	return botResp, paginatedResponse, http.StatusOK, nil
}
