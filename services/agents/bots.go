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

func FetchOrganisationBots(db *gorm.DB, logger *utility.Logger, org_id string, c *gin.Context, extReq request.ExternalRequest, redisClient *redis.Client) (models.AgentsResp, postgresql.PaginationResponse, int, error) {
	var (
		orgInt  models.OrganisationIntegrations
		botResp models.AgentsResp = make(models.AgentsResp, 0)
		seenBot                   = make(map[string]bool)
	)

	resp, paginatedResponse, err, code := orgInt.GetCustomAgentApp(db, org_id, c)
	if err != nil {
		return models.AgentsResp{}, paginatedResponse, code, err
	}

	for _, org_agents := range resp {
		json_url := org_agents.JSONUrl

		if seenBot[json_url] {
			continue
		}
		seenBot[json_url] = true

		data_r, err := models.FetchDetailsFromAgentJSON(extReq, json_url, redisClient)
		if err != nil {
			// logger.Error("failed to fetch agent json", err)

			// failedAgent := models.Integrations{
			// 	ID:             org_agents.IntegrationID,
			// 	Name:           "Failed Bot",
			// 	JSONUrl:        org_agents.JSONUrl,
			// 	AppUrl:         "Failed to fetch app url",
			// 	AppLogo:        "Failed to fetch app logo",
			// 	AppDescription: "Failed to fetch app description",
			// 	Category:       "Failed to fetch category",
			// 	Status:         "failed",
			// 	IsActive:       org_agents.IsActive,
			// 	CreatedAt:      org_agents.CreatedAt,
			// 	UpdatedAt:      org_agents.UpdatedAt,
			// }
			// botResp = append(botResp, struct {
			// 	models.Integrations
			// 	Linked bool "json:\"linked\""
			// }{
			// 	Integrations: failedAgent,
			// 	Linked:       false,
			// })
			continue
		}

		description, ok := data_r["descriptions"].(map[string]interface{})
		if !ok {
			logger.Error("failed to fetch agent json description", err)
			continue
		}

		appName, ok := description["app_name"].(string)
		if !ok || appName == "" {
			appName = "Undefined"
		}

		category, ok := data_r["integration_category"].(string)
		if !ok || category == "" {
			category = "Undefined"
		}

		is_bot, ok := data_r["bot"].(bool)
		if !ok {
			is_bot = false
		}

		if is_bot {
			appUrl, _ := description["app_url"].(string)
			appLogo, _ := description["app_logo"].(string)
			appDescription, _ := description["app_description"].(string)

			agent := models.Integrations{
				ID:             org_agents.IntegrationID,
				Name:           appName,
				JSONUrl:        org_agents.JSONUrl,
				AppUrl:         appUrl,
				AppLogo:        appLogo,
				AppDescription: appDescription,
				Category:       category,
				Status:         "success",
				IsActive:       org_agents.IsActive,
				CreatedAt:      org_agents.CreatedAt,
				UpdatedAt:      org_agents.UpdatedAt,
			}

			botResp = append(botResp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: agent,
				Linked:       true,
			})
		}

	}

	return botResp, paginatedResponse, http.StatusOK, nil
}
