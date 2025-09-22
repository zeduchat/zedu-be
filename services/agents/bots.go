package agents

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func FetchOrganisationAgents(db *gorm.DB, logger *utility.Logger, org_id string, c *gin.Context, extReq request.ExternalRequest, redisClient *redis.Client) ([]models.AgentResp, postgresql.PaginationResponse, int, error) {
	var (
		orgInt  models.OrganisationIntegrations
		botResp []models.AgentResp = make([]models.AgentResp, 0)
	)

	resp, paginatedResponse, err, code := orgInt.GetCustomAgentApps(db, org_id, c)
	if err != nil {
		return []models.AgentResp{}, paginatedResponse, code, err
	}

	for _, org_agents := range resp {
		parts := strings.Split(org_agents.IntegrationID, "-")
		lastPart := parts[len(parts)-1]

		agent := models.AgentResp{
			ID:          org_agents.IntegrationID,
			Name:        org_agents.AppName,
			Title:       org_agents.Title,
			Tone:        org_agents.Tone,
			Visibility:  org_agents.Visibility,
			Avatar:      org_agents.AppLogo,
			Description: org_agents.AppDescription,
			IsActive:    org_agents.IsActive,
			AgentSlug:   fmt.Sprintf("%s-%s", slug.Make(org_agents.AppName), lastPart),
		}

		botResp = append(botResp, agent)
	}

	return botResp, paginatedResponse, http.StatusOK, nil
}
