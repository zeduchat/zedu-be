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

func FetchOrganisationAgents(db *gorm.DB, logger *utility.Logger, ids models.IDS, c *gin.Context, extReq request.ExternalRequest, redisClient *redis.Client) ([]models.AgentResp, postgresql.PaginationResponse, int, error) {
	var (
		orgInt models.OrganisationIntegrations
	)

	resp, paginatedResponse, err, code := orgInt.GetCustomAgentApps(db, ids, c)
	if err != nil {
		return []models.AgentResp{}, paginatedResponse, code, err
	}

	for id, org_agents := range resp {
		parts := strings.Split(org_agents.ID, "-")
		lastPart := parts[len(parts)-1]

		resp[id].AgentSlug = fmt.Sprintf("%s-%s", slug.Make(org_agents.Name), lastPart)
	}

	return resp, paginatedResponse, http.StatusOK, nil
}
