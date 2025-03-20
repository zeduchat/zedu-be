package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func FetchOrganisationBots(db *gorm.DB, logger *utility.Logger, org_id string,c *gin.Context, extReq request.ExternalRequest) (models.AgentsResp, postgresql.PaginationResponse, int, error) {
	var (
		orgInt models.OrganisationIntegrations
		botResp models.AgentsResp
	)

	resp, paginatedResponse, err, code := orgInt.GetCustomAgentApp(db, org_id, c)
	if err != nil {
		return models.AgentsResp{}, paginatedResponse, code, err
	}

	for _, org_agents := range resp {
		json_url := org_agents.JSONUrl
		data := map[string]string{"url": json_url}

		response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)
		if err != nil {
			continue
		}

		response_data := response.(map[string]interface{})
		data_r := response_data["data"].(map[string]interface{})
		description := data_r["descriptions"].(map[string]interface{})
		category, ok := data_r["agent_category"].(string)
		if !ok || category == "" {
			category = "Undefined"
		}
		is_bot, ok := data_r["bot"].(bool)
		if !ok {
			is_bot = false
		}

		if is_bot{
			agent := models.Integrations{
				ID:             org_agents.IntegrationID,
				Name:           description["app_name"].(string),
				JSONUrl:        org_agents.JSONUrl,
				AppUrl:         description["app_url"].(string),
				AppLogo:        description["app_logo"].(string),
				AppDescription: description["app_description"].(string),
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
