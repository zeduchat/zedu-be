package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func GetAllAgentApp(c *gin.Context, org_id string, db *gorm.DB) (models.AgentsResp, error) {
	var agents models.Integrations

	resp, err := agents.GetAllAgentApp(db, org_id, c)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func GetCustomAgentApp(c *gin.Context, org_id string, db *gorm.DB, extReq request.ExternalRequest) (models.AgentsResp, postgresql.PaginationResponse, error, int) {
	var org_agents models.OrganisationIntegrations

	var int_resp = models.AgentsResp{}

	resp, paginationResult, err, code := org_agents.GetCustomAgentApp(db, org_id, c)

	if err != nil {
		return nil, postgresql.PaginationResponse{}, err, code
	}

	for _, org_agents := range resp {

		json_url := org_agents.JSONUrl
		data := map[string]string{"url": json_url}

		response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

		if err != nil {
			agent := models.Integrations{
				ID:             org_agents.IntegrationID,
				Name:           "Unavailable",
				JSONUrl:        org_agents.JSONUrl,
				AppDescription: "This agent is currently unavailable.",
				Category:       "Unavailable",
				IsActive:       false,
				Status:         "failed",
				CreatedAt:      org_agents.CreatedAt,
				UpdatedAt:      org_agents.UpdatedAt,
			}

			int_resp = append(int_resp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: agent,
				Linked:       true,
			})
			continue
		}

		response_data := response.(map[string]interface{})

		data_r := response_data["data"].(map[string]interface{})

		description := data_r["descriptions"].(map[string]interface{})
		category, ok := data_r["integration_category"].(string)

		if !ok || category == "" {
			category = "Undefined"
		}

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

		int_resp = append(int_resp, struct {
			models.Integrations
			Linked bool "json:\"linked\""
		}{
			Integrations: agent,
			Linked:       true,
		})
	}

	return int_resp, paginationResult, nil, code
}

func GetSystemAgentApps(c *gin.Context, db *gorm.DB, extReq request.ExternalRequest) (models.AgentsResp, postgresql.PaginationResponse, error, int) {
	var agents models.Integrations

	var int_resp = models.AgentsResp{}

	resp, paginationResult, err, code := agents.GetSystemAgentApps(db, c)

	if err != nil {
		return nil, postgresql.PaginationResponse{}, err, code
	}

	for _, org_agents := range resp {

		json_url := org_agents.JSONUrl
		data := map[string]string{"url": json_url}

		response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

		if err != nil {
			agent := models.Integrations{
				ID:             org_agents.ID,
				Name:           "Unavailable",
				JSONUrl:        org_agents.JSONUrl,
				AppDescription: "This agent is currently unavailable.",
				Category:       "Unavailable",
				IsActive:       false,
				Status:         "failed",
				CreatedAt:      org_agents.CreatedAt,
				UpdatedAt:      org_agents.UpdatedAt,
			}

			int_resp = append(int_resp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: agent,
				Linked:       true,
			})
			continue
		}

		response_data := response.(map[string]interface{})

		data_r := response_data["data"].(map[string]interface{})

		description := data_r["descriptions"].(map[string]interface{})

		category, ok := data_r["integration_category"].(string)

		info, ok := data_r["info"].(string)
		if !ok || info == "" {
			info = "Undefined"
		}

		if !ok || category == "" {

			category = "Undefined"
		}

		agent := models.Integrations{
			ID:             org_agents.ID,
			Name:           description["app_name"].(string),
			JSONUrl:        org_agents.JSONUrl,
			AppUrl:         description["app_url"].(string),
			AppLogo:        description["app_logo"].(string),
			AppDescription: description["app_description"].(string),
			Info:           info,
			Category:       category,
			Status:         "success",
			IsActive:       org_agents.IsActive,
			CreatedAt:      org_agents.CreatedAt,
			UpdatedAt:      org_agents.UpdatedAt,
		}

		int_resp = append(int_resp, struct {
			models.Integrations
			Linked bool "json:\"linked\""
		}{
			Integrations: agent,
			Linked:       true,
		})
	}

	return int_resp, paginationResult, nil, code
}

func GetSystemAgentApp(c *gin.Context, db *gorm.DB, int_id string, extReq request.ExternalRequest) (models.Integrations, error, int) {
	var agents models.Integrations

	resp, err, code := agents.GetSystemAgentApp(db, int_id, c)

	if err != nil {
		return models.Integrations{}, err, code
	}

	json_url := resp.JSONUrl
	data := map[string]string{"url": json_url}

	response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

	if err != nil {
		extReq.Logger.Error("An error occurred while fetching agent json, err: %s ", err)
		agent := models.Integrations{
			ID:             resp.ID,
			Name:           "Unavailable",
			JSONUrl:        resp.JSONUrl,
			AppDescription: "This agent is currently unavailable.",
			Category:       "Unavailable",
			IsActive:       false,
			Status:         "failed",
			CreatedAt:      resp.CreatedAt,
			UpdatedAt:      resp.UpdatedAt,
		}

		return agent, nil, code
	}

	response_data := response.(map[string]interface{})

	data_r := response_data["data"].(map[string]interface{})

	description := data_r["descriptions"].(map[string]interface{})

	info, ok := data_r["info"].(string)
	if !ok {
		info = "Undefined"
	}

	agent := models.Integrations{
		ID:             resp.ID,
		Name:           description["app_name"].(string),
		JSONUrl:        resp.JSONUrl,
		Status:         "success",
		AppUrl:         description["app_url"].(string),
		AppLogo:        description["app_logo"].(string),
		AppDescription: description["app_description"].(string),
		Info:           info,
		IsActive:       resp.IsActive,
		CreatedAt:      resp.CreatedAt,
		UpdatedAt:      resp.UpdatedAt,
	}

	return agent, nil, code
}

func UpdateAgentApp(req models.UpdateAgent, ids map[string]string, db *gorm.DB) (models.Integrations, error) {
	var agent models.Integrations

	updatedAgent, err := agent.UpdateAgent(db, ids, req)
	if err != nil {
		return models.Integrations{}, err
	}

	return updatedAgent, nil
}

func DeleteAgentApp(ids map[string]string, db *gorm.DB) error {
	var agent models.Integrations

	err := agent.DeleteAgent(db, ids)
	if err != nil {
		return err
	}

	return nil
}

// Delete Org Custom Integration
func DeleteCustomAgentApp(ids map[string]string, db *gorm.DB) (error, int) {
	var org_agent models.OrganisationIntegrations

	err, code := org_agent.DeleteCustomAgent(db, ids)
	if err != nil {
		return err, code
	}

	return nil, code
}

func ChangeAgentStatus(ids map[string]string, req models.ChangeAgentStatus, db *gorm.DB, extReq request.ExternalRequest) error {
	var agent models.OrganisationIntegrations

	err := agent.ChangeStatus(db, req, ids, extReq)
	if err != nil {
		return err
	}

	return nil
}

func ChangeAgentSendBackStatus(ids map[string]string, req models.ChangeAgentStatus, db *gorm.DB) error {
	var agent models.OrganisationChannelsIntegrations

	err := agent.ChangeSendBackStatus(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}

func UpdateJSONSchema(ids map[string]string, req models.UpdateJSONSchemaRequest, db *gorm.DB) error {
	var orgIntegration models.OrganisationIntegrations

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return errors.New("organisation does not have that agent")
	}

	err := orgIntegration.UpdateJSONSchema(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}

func CreateCustomAgent(org_id string, req models.CustomIntegrationRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var (
		orgIntegration models.OrganisationIntegrations
		agentSettings  models.CustomIntegrationsSetting
	)

	data := map[string]string{"url": req.JSONUrl}

	response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

	if err != nil {
		return errors.New("Failed to create custom agent, invalid JSON supplied")
	}

	response_data := response.(map[string]interface{})
	data_r, ok := response_data["data"].(map[string]interface{})

	if !ok {
		return errors.New("Failed to Create Custom Integration, data field does not exist")
	}

	// validate description entry

	err = models.ValidateAgentData(data_r)

	if err != nil {
		return err
	}

	settings, ok := data_r["settings"]
	if !ok {
		return errors.New("Failed to create custom agent, settings field does not exist")
	}

	settings_data := map[string]interface{}{"settings": settings}

	// create agent in db
	orgIntegration.OrgID = org_id
	orgIntegration.JSONUrl = req.JSONUrl
	orgIntegration.IntegrationID = utility.GenerateUUID()
	orgIntegration.IsActive = false
	orgIntegration.IsSystem = false
	orgIntegration.ID = utility.GenerateUUID()

	err = orgIntegration.CreateOrganisationIntegration(db)

	if err != nil {
		return err
	}

	is_auth, ok := data_r["is_oauth"].(bool)

	if ok && is_auth {
		enc_key := config.Config.Server.EncKey

		api_key, err := utility.CreateExternalApiKey(org_id, orgIntegration.IntegrationID, enc_key)

		auth_credentials := map[string]interface{}{"agent_auth_credentials": "Not-Set-Yet"}

		auth_credentials["telex_api_key"] = api_key
		settings_data["auth_credentials"] = auth_credentials

		if err != nil {
			return errors.New("Failed to create external API key")
		}
	}

	// serialize the settings json

	settingJsonData, err := json.Marshal(settings_data)
	if err != nil {
		return fmt.Errorf("error serializing to JSON: %v", err)
	}

	serialized_settings := string(settingJsonData)

	agentSettings.ID = utility.GenerateUUID()
	agentSettings.SettingEntry = serialized_settings
	agentSettings.OrgID = org_id
	agentSettings.IsSystem = false
	agentSettings.IntegrationID = orgIntegration.IntegrationID

	err = agentSettings.CreateIntegrationSettings(db)

	if err != nil {
		return errors.New("Failed to create agent settings")
	}

	return nil
}

// Update CustomIntegration
func UpdateCustomAgent(ids map[string]string, req models.CustomIntegrationRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var orgIntegration models.OrganisationIntegrations

	data := map[string]string{"url": req.JSONUrl}

	_, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

	if err != nil {
		return errors.New("Failed to Update Custom Integration, invalid JSON supplied")
	}

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return errors.New("organisation does not have that agent")
	}

	err = orgIntegration.UpdateCustomIntegration(db, req, ids)

	if err != nil {
		return err
	}

	return nil
}

func GetOrganisationChannelAgents(db *gorm.DB, channel_id, org_id string, c *gin.Context, extReq request.ExternalRequest) (models.AgentsResp, postgresql.PaginationResponse, int, error) {
	var (
		ocIntegrations models.OrganisationChannelsIntegrations
	)

	var int_resp = models.AgentsResp{}

	agents, paginationResponse, code, err := ocIntegrations.GetOrganisationChannelAgents(db, channel_id, org_id, c)

	if err != nil {
		return nil, paginationResponse, code, err
	}

	for _, org_agents := range agents {

		json_url := org_agents.JSONUrl
		data := map[string]string{"url": json_url}

		response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

		if err != nil {

			agent := models.Integrations{
				ID:             org_agents.IntegrationID,
				Name:           "Unavailable",
				JSONUrl:        org_agents.JSONUrl,
				AppDescription: "This agent is currently unavailable.",
				Category:       "Unavailable",
				IsActive:       false,
				Status:         "failed",
				CreatedAt:      org_agents.CreatedAt,
				UpdatedAt:      org_agents.UpdatedAt,
			}

			int_resp = append(int_resp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: agent,
				Linked:       true,
			})

			continue
		}

		response_data := response.(map[string]interface{})

		data_r := response_data["data"].(map[string]interface{})

		description := data_r["descriptions"].(map[string]interface{})

		category, ok := data_r["agent_category"].(string)

		if !ok || category == "" {

			category = "Undefined"
		}

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

		int_resp = append(int_resp, struct {
			models.Integrations
			Linked bool "json:\"linked\""
		}{
			Integrations: agent,
			Linked:       true,
		})
	}

	return int_resp, paginationResponse, code, nil
}

func ActivateChannelAgent(ids map[string]string, req models.ActivateChannelAgent, db *gorm.DB) error {
	var (
		ocIntegrations  models.OrganisationChannelsIntegrations
		orgIntegrations models.OrganisationIntegrations
		channels        models.Channels
	)

	exists := postgresql.CheckExists(db, &orgIntegrations, "org_id = ? AND integration_id = ?", ids["organisation_id"], ids["agent_id"])
	if !exists {
		return errors.New("organisation does not have that agent")
	}

	exists = postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", ids["channel_id"], ids["organisation_id"])
	if !exists {
		return errors.New("organisation does not have that channel")
	}

	err := ocIntegrations.ActivateChannelAgent(db, req, ids)
	if err != nil {
		return err
	}

	return nil
}

func AgentChannels(ids map[string]string, db *gorm.DB) (gin.H, error) {
	var (
		ocIntegrations  models.OrganisationChannelsIntegrations
		orgIntegrations models.OrganisationIntegrations
		res             gin.H
	)

	exists := postgresql.CheckExists(db, &orgIntegrations, "org_id = ? AND integration_id = ?", ids["organisation_id"], ids["agent_id"])
	if !exists {
		return nil, errors.New("organisation does not have that agent")
	}

	response, is_allChannels, err := ocIntegrations.FetchIntegrationChannels(db, ids)
	if err != nil {
		return res, err
	}

	res = gin.H{
		"is_allchannels": !is_allChannels,
		"channels":       response,
	}

	return res, nil
}

func CheckAgentIsActive(ids map[string]string, db *gorm.DB) (gin.H, error) {
	var (
		ocIntegrations  models.OrganisationChannelsIntegrations
		orgIntegrations models.OrganisationIntegrations
		res             gin.H
	)

	exists := postgresql.CheckExists(db, &orgIntegrations, "org_id = ? AND integration_id = ?", ids["organisation_id"], ids["agent_id"])
	if !exists {
		return nil, errors.New("organisation does not exist or have that agent")
	}

	status, err := ocIntegrations.CheckIntegrationIsActive(db, ids)
	if err != nil {
		return res, err
	}

	res = gin.H{
		"status": status,
	}

	return res, nil
}

func UpdateCustomAgentSettings(ids map[string]string, req models.CustomIntegrationSettingRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var (
		orgIntegration models.OrganisationIntegrations
		ucis           models.CustomIntegrationsSetting
	)

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return errors.New("organisation does not exist or have that agent")
	}

	settings := req.SettingEntry

	_, ok := settings["settings"]
	if !ok {

		return fmt.Errorf("settings field is required")

	}
	settingJsonData, err := json.Marshal(settings)

	if err != nil {
		return fmt.Errorf("error serializing to JSON: %v", err)
	}

	serialized_settings := string(settingJsonData)
	req.SerializedEntry = serialized_settings

	err = ucis.UpdateCustomIntegrationSettings(db, req, ids)

	if err != nil {
		return err
	}

	return nil
}

func GetCustomAgentSettings(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]interface{}, int, error) {

	var (
		orgIntegration models.OrganisationIntegrations
		ucis           models.CustomIntegrationsSetting

		deserialize_settings map[string]interface{}
		resp                 map[string]interface{}
	)

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return deserialize_settings, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	exists = postgresql.CheckExists(db, &ucis, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return deserialize_settings, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	settings := ucis.SettingEntry

	// unserialize the settings text

	err := json.Unmarshal([]byte(settings), &deserialize_settings)

	if err != nil {
		return resp, http.StatusInternalServerError, fmt.Errorf("Error deserializing JSON: %v", err)
	}

	resp = make(map[string]interface{})

	resp["is_system"] = ucis.IsSystem
	resp["is_active"] = orgIntegration.IsActive
	resp["settings"] = deserialize_settings["settings"]

	return resp, http.StatusOK, nil
}

func GetCustomAgentStatus(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]interface{}, int, error) {

	var (
		orgIntegration       models.OrganisationIntegrations
		ucis                 models.CustomIntegrationsSetting
		deserialize_settings map[string]interface{}
	)
	status := make(map[string]interface{})

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return status, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	exists = postgresql.CheckExists(db, &ucis, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return deserialize_settings, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	err := json.Unmarshal([]byte(ucis.SettingEntry), &deserialize_settings)

	if err != nil {
		return status, http.StatusInternalServerError, fmt.Errorf("Error deserializing JSON: %v", err)
	}

	status["is_system"] = orgIntegration.IsSystem
	status["is_active"] = orgIntegration.IsActive

	auth_credentials, ok := deserialize_settings["auth_credentials"].(map[string]interface{})

	if ok {
		api_key, ok := auth_credentials["telex_api_key"].(string)

		if ok && api_key != "" {
			status["telex_api_key"] = api_key
		}
	}

	return status, http.StatusOK, nil
}

// Integration External Requests

func GetCustomAgentSettingsExteranl(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]interface{}, int, error) {

	var (
		ucis models.CustomIntegrationsSetting

		deserialize_settings map[string]interface{}
	)

	exists := postgresql.CheckExists(db, &ucis, "org_id::text LIKE ? AND integration_id::text LIKE ?", "%"+ids["porg_id"], "%"+ids["pagent_id"])
	if !exists {
		return deserialize_settings, http.StatusNotFound, errors.New("Integration not connnected yet")
	}

	settings := ucis.SettingEntry

	// unserialize the settings text

	err := json.Unmarshal([]byte(settings), &deserialize_settings)

	if err != nil {
		return deserialize_settings, http.StatusInternalServerError, fmt.Errorf("Error deserializing JSON: %v", err)
	}

	return deserialize_settings, http.StatusOK, nil
}

func UpdateCustomAgentSettingsExternal(ids map[string]string, req models.CustomIntegrationSettingRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var (
		orgIntegration       models.OrganisationIntegrations
		ucis                 models.CustomIntegrationsSetting
		deserialize_settings map[string]interface{}
	)

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id::text LIKE ? AND integration_id::text LIKE ?", "%"+ids["porg_id"], "%"+ids["pagent_id"])
	if !exists {
		return errors.New("integration not connected yet")
	}

	exists = postgresql.CheckExists(db, &ucis, "org_id::text LIKE ? AND integration_id::text LIKE ?", "%"+ids["porg_id"], "%"+ids["pagent_id"])
	if !exists {
		return errors.New("integration not connnected yet")
	}

	db_settings := ucis.SettingEntry

	// unserialize the settings text

	err := json.Unmarshal([]byte(db_settings), &deserialize_settings)

	if err != nil {
		return fmt.Errorf("error deserializing JSON")
	}

	auth_credentials, ok := deserialize_settings["auth_credentials"].(map[string]interface{})

	if ok {
		api_key, ok := auth_credentials["telex_api_key"].(string)
		if ok && api_key != ids["telex_api_key"] {
			return errors.New("an error occured: api_key Mismatch")
		}
	}

	settings := req.SettingEntry
	settingJsonData, err := json.Marshal(settings)

	if err != nil {
		return fmt.Errorf("error serializing to JSON")
	}

	serialized_settings := string(settingJsonData)
	req.SerializedEntry = serialized_settings

	ids["org_id"] = ucis.OrgID
	ids["agent_id"] = ucis.IntegrationID

	err = ucis.UpdateCustomIntegrationSettings(db, req, ids)

	if err != nil {
		return err
	}

	return nil
}
