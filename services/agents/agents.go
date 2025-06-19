package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
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

		if org_agents.AppName == "" {

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
				Version:        org_agents.Version,
				Prices:         org_agents.Prices,
				Provider:       org_agents.Provider,
				IsPaid:         org_agents.IsPaid,
				IsApproved:     org_agents.IsApproved,
				Skills:         org_agents.Skills,
			}

			int_resp = append(int_resp, struct {
				models.Integrations
				Linked bool "json:\"linked\""
			}{
				Integrations: agent,
				Linked:       true,
			})

			err = org_agents.UpdateCustomIntegration(db, models.CustomIntegrationRequest{
				AppName:        description["app_name"].(string),
				JSONUrl:        org_agents.JSONUrl,
				AppUrl:         description["app_url"].(string),
				AppLogo:        description["app_logo"].(string),
				AppDescription: description["app_description"].(string),
			}, map[string]string{"agent_id": org_agents.IntegrationID})
			if err != nil {
				extReq.Logger.Error("an error occurred while saving agent details, %v", err)
			}
			continue
		}

		agent := models.Integrations{
			ID:             org_agents.IntegrationID,
			Name:           org_agents.AppName,
			JSONUrl:        org_agents.JSONUrl,
			AppUrl:         org_agents.AppUrl,
			AppLogo:        org_agents.AppLogo,
			AppDescription: org_agents.AppDescription,
			Category:       "Agents",
			Status:         "success",
			IsActive:       org_agents.IsActive,
			CreatedAt:      org_agents.CreatedAt,
			UpdatedAt:      org_agents.UpdatedAt,
			Version:        org_agents.Version,
			Prices:         org_agents.Prices,
			Provider:       org_agents.Provider,
			IsPaid:         org_agents.IsPaid,
			IsApproved:     org_agents.IsApproved,
			Skills:         org_agents.Skills,
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
func DeleteCustomAgentApp(db *gorm.DB, logger utility.Logger, ids models.IDS) (error, int) {
	var org_agent models.OrganisationIntegrations

	err, code := org_agent.DeleteCustomAgent(db, logger, ids)
	if err != nil {
		return err, code
	}

	return nil, code
}

func ChangeStatus(ids map[string]string, req models.ChangeAgentStatus, db *gorm.DB, extReq request.ExternalRequest) error {

	// if req.Status {
	// 	return SendAgentApiKey(ids, req, db, extReq)
	// }

	var (
		orgIntegration models.OrganisationIntegrations
	)

	err := orgIntegration.ChangeStatus(db, req, ids, extReq)
	if err != nil {
		return err
	}

	return nil
}

func SendAgentApiKey(ids map[string]string, req models.ChangeAgentStatus, db *gorm.DB, extReq request.ExternalRequest) error {
	var (
		orgIntegration       models.OrganisationIntegrations
		ucis                 models.CustomIntegrationsSetting
		deserialize_settings map[string]interface{}
		api_key              string
	)

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id =  ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return errors.New("integration not connected yet")
	}

	exists = postgresql.CheckExists(db, &ucis, "org_id = ? AND integration_id =  ?", ids["org_id"], ids["agent_id"])
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
		api_key, _ = auth_credentials["agent_api_key"].(string)
	}

	// send api key to agent

	parsedURL, _ := url.Parse(orgIntegration.JSONUrl)
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	dataPayload := map[string]interface{}{
		"url": baseURL,
		"payload": map[string]string{
			"org_id":  ids["org_id"],
			"api_key": api_key,
		}}

	response, err := extReq.SendExternalRequest(request.SendAgentAPIKey, dataPayload)

	if err != nil {
		extReq.Logger.Error("An error occured: %v", response)
		return errors.New("failed to activate agent, an error occured")
	}

	extReq.Logger.Info("Request sent to agent: %v", response)

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

func CreateCustomAgent(org_id string, req models.CustomIntegrationRequest, db *gorm.DB, extReq request.ExternalRequest, user_id string) (models.AgentResp, error) {
	var (
		orgIntegration models.OrganisationIntegrations
		agentSettings  models.CustomIntegrationsSetting
		organisation   models.Organisation
		int_resp       models.AgentResp
		org            models.Organisation
	)

	isMember, err := org.CheckUserIsMemberOfOrg(user_id, org_id, db)
	if err != nil {
		return int_resp, err
	}
	if !isMember {
		return int_resp, errors.New("user not a member of organisation")
	}

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", org_id)
	if !organisationExists {
		return int_resp, errors.New("organisation does not exist")
	}

	err = validateJSONURL(req.JSONUrl)
	if err != nil {
		return int_resp, err
	}

	agentID, err := utility.GenerateUUIDFromString(req.JSONUrl)
	if err != nil {
		return int_resp, fmt.Errorf("error generating agent ID from JSON URL: %v", err)
	}

	// Check if the agent already exists in the organization
	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", org_id, agentID)
	if exists {
		return int_resp, errors.New("organisation already has that agent")
	}

	// Make the external request to the JSON URL
	data := map[string]string{"url": req.JSONUrl}
	response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

	if err != nil {
		return int_resp, errors.New("failed to create agent, invalid JSON supplied")
	}

	data_r, ok := response.(map[string]interface{})
	// data_r, ok := response_data["data"].(map[string]interface{})
	if !ok {
		return int_resp, errors.New("failed to create agent, data field does not exist")
	}

	err = models.ValidateAgentData(data_r)
	if err != nil {
		return int_resp, err
	}

	psk, err := models.GenerateAgentKey()
	if err != nil {
		return int_resp, err
	}

	bytes, err := json.Marshal(data_r)
	if err != nil {
		return int_resp, err
	}

	var payload models.OrganisationIntegrations
	json.Unmarshal(bytes, &payload)

	settings := ""

	settings_data := map[string]any{"settings": settings}

	orgIntegration.OrgID = org_id
	orgIntegration.JSONUrl = req.JSONUrl
	orgIntegration.IntegrationID = agentID
	orgIntegration.IsActive = true
	orgIntegration.IsSystem = false
	orgIntegration.ID = utility.GenerateUUID()
	orgIntegration.AppName = data_r["name"].(string)
	orgIntegration.AppDescription = data_r["description"].(string)
	orgIntegration.AppUrl = data_r["url"].(string)
	orgIntegration.Prices = payload.Prices
	orgIntegration.Provider = payload.Provider
	orgIntegration.Version = payload.Version
	orgIntegration.DefaultInputModes = payload.DefaultInputModes
	orgIntegration.DefaultOutputModes = payload.DefaultOutputModes
	orgIntegration.Skills = payload.Skills
	orgIntegration.IsPaid = payload.IsPaid
	orgIntegration.PreSharedKey = psk
	orgIntegration.OwnerID = user_id

	err = orgIntegration.CreateOrganisationIntegration(db)
	if err != nil {
		return int_resp, err
	}

	auth_credentials := map[string]any{"agent_auth_credentials": "Not-Set-Yet"}
	auth_credentials["agent_api_key"] = psk
	settings_data["auth_credentials"] = auth_credentials

	settingJsonData, err := json.Marshal(settings_data)
	if err != nil {
		return int_resp, fmt.Errorf("error serializing to JSON: %v", err)
	}
	serialized_settings := string(settingJsonData)

	agentSettings.ID = utility.GenerateUUID()
	agentSettings.SettingEntry = serialized_settings
	agentSettings.OrgID = org_id
	agentSettings.IsSystem = false
	agentSettings.IntegrationID = orgIntegration.IntegrationID

	err = agentSettings.CreateIntegrationSettings(db)
	if err != nil {
		return int_resp, errors.New("failed to create agent settings")
	}

	agent := models.Integrations{
		ID:             orgIntegration.IntegrationID,
		Name:           orgIntegration.AppName,
		JSONUrl:        orgIntegration.JSONUrl,
		AppUrl:         orgIntegration.AppUrl,
		AppLogo:        orgIntegration.AppLogo,
		AppDescription: orgIntegration.AppDescription,
		Category:       "Agents",
		Status:         "success",
		IsActive:       orgIntegration.IsActive,
		CreatedAt:      orgIntegration.CreatedAt,
		UpdatedAt:      orgIntegration.UpdatedAt,
	}

	int_resp = struct {
		models.Integrations
		Linked bool "json:\"linked\""
	}{
		Integrations: agent,
		Linked:       true,
	}

	return int_resp, nil
}

func validateJSONURL(jsonUrl string) error {
	validator := utility.NewURLValidator()
	if err := validator.Validate(jsonUrl); err != nil {
		return fmt.Errorf("invalid JSON URL: %v", err)
	}
	return nil
}

// Update CustomIntegration
func UpdateCustomAgent(ids map[string]string, req models.CustomIntegrationRequest, db *gorm.DB, extReq request.ExternalRequest) error {

	var orgIntegration models.OrganisationIntegrations

	data := map[string]string{"url": req.JSONUrl}

	_, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

	if err != nil {
		return errors.New("failed to Update Custom Integration, invalid JSON supplied")
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

		if org_agents.AppName == "" {

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

			err = org_agents.UpdateCustomIntegration(db, models.CustomIntegrationRequest{
				AppName:        description["app_name"].(string),
				JSONUrl:        org_agents.JSONUrl,
				AppUrl:         description["app_url"].(string),
				AppLogo:        description["app_logo"].(string),
				AppDescription: description["app_description"].(string),
			}, map[string]string{"agent_id": org_agents.IntegrationID})
			if err != nil {
				extReq.Logger.Error("an error occurred while saving agent details, %v", err)
			}
			continue
		}

		agent := models.Integrations{
			ID:             org_agents.IntegrationID,
			Name:           org_agents.AppName,
			JSONUrl:        org_agents.JSONUrl,
			AppUrl:         org_agents.AppUrl,
			AppLogo:        org_agents.AppLogo,
			AppDescription: org_agents.AppDescription,
			Category:       "Agents",
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
		api_key, ok := auth_credentials["agent_api_key"].(string)

		if ok && api_key != "" && ids["user_id"] == orgIntegration.OwnerID {
			status["agent_api_key"] = api_key
		} else {
			status["agent_api_key"] = ""
		}
	}

	return status, http.StatusOK, nil
}

// Integration External Requests

func GetCustomAgentSettingsExteranl(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]interface{}, int, error) {
	var (
		ucis                 models.CustomIntegrationsSetting
		deserialize_settings map[string]interface{}
	)

	err := db.Where("org_id = ? AND integration_id = ?", ids["porg_id"], ids["pagent_id"]).First(&ucis).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return deserialize_settings, http.StatusNotFound, errors.New("integration not connected yet")
		}
		return deserialize_settings, http.StatusInternalServerError, fmt.Errorf("error fetching agent settings: %v", err)
	}

	// Unserialize the settings text
	err = json.Unmarshal([]byte(ucis.SettingEntry), &deserialize_settings)
	if err != nil {
		return deserialize_settings, http.StatusInternalServerError, fmt.Errorf("error deserializing JSON: %v", err)
	}

	return deserialize_settings, http.StatusOK, nil
}

func UpdateCustomAgentSettingsExternal(ids map[string]string, req models.CustomIntegrationSettingRequest, db *gorm.DB, extReq request.ExternalRequest) error {
	var (
		orgIntegration       models.OrganisationIntegrations
		ucis                 models.CustomIntegrationsSetting
		deserialize_settings map[string]interface{}
	)

	err := db.Where("org_id = ? AND integration_id = ?", ids["porg_id"], ids["pagent_id"]).First(&orgIntegration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("integration not connected yet")
		}
		return fmt.Errorf("error fetching organisation integration: %v", err)
	}

	err = db.Where("org_id = ? AND integration_id = ?", ids["porg_id"], ids["pagent_id"]).First(&ucis).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("integration not connected yet")
		}
		return fmt.Errorf("error fetching custom integration settings: %v", err)
	}

	db_settings := ucis.SettingEntry

	// Unserialize the settings JSON
	err = json.Unmarshal([]byte(db_settings), &deserialize_settings)
	if err != nil {
		return fmt.Errorf("error deserializing JSON")
	}

	// Check for agent_api_key match
	auth_credentials, ok := deserialize_settings["auth_credentials"].(map[string]interface{})
	if ok {
		api_key, ok := auth_credentials["agent_api_key"].(string)
		if ok && api_key != ids["agent_api_key"] {
			return errors.New("an error occurred: api_key mismatch")
		}
	}

	settings := req.SettingEntry
	settingJsonData, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("error serializing to JSON")
	}
	req.SerializedEntry = string(settingJsonData)

	ids["org_id"] = ucis.OrgID
	ids["agent_id"] = ucis.IntegrationID
	err = ucis.UpdateCustomIntegrationSettings(db, req, ids)
	if err != nil {
		return err
	}

	// Change agent status
	reqStatus := models.ChangeAgentStatus{Status: true}
	err = orgIntegration.ChangeStatus(db, reqStatus, ids, extReq)
	if err != nil {
		return err
	}

	return nil
}

func AgentCallback(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) error {
	var orgIntegration models.OrganisationIntegrations

	err := db.Where("org_id = ? AND integration_id = ?", ids["porg_id"], ids["pagent_id"]).First(&orgIntegration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("integration not connected yet")
		}
		return fmt.Errorf("error fetching integration: %v", err)
	}

	ids["org_id"] = orgIntegration.OrgID
	ids["agent_id"] = orgIntegration.IntegrationID

	// Change status
	reqStatus := models.ChangeAgentStatus{Status: true}
	if err := orgIntegration.ChangeStatus(db, reqStatus, ids, extReq); err != nil {
		return err
	}

	return nil
}

func GetAllCustomAgent(c *gin.Context, db *gorm.DB) (models.AgentsResp, postgresql.PaginationResponse, error, int) {
	var org_agents models.OrganisationIntegrations

	var int_resp = models.AgentsResp{}

	resp, paginationResult, err, code := org_agents.GetAllCustomAgent(db, c)

	if err != nil {
		return nil, postgresql.PaginationResponse{}, err, code
	}

	for _, org_agents := range resp {

		agent := models.Integrations{
			ID:             org_agents.ID,
			Name:           org_agents.Name,
			AppUrl:         org_agents.AppUrl,
			AppLogo:        org_agents.AppLogo,
			AppDescription: org_agents.AppDescription,
			Category:       "Agents",
			Status:         "success",
			IsActive:       org_agents.IsActive,
			Provider:       org_agents.Provider,
			CreatedAt:      org_agents.CreatedAt,
			Version:        org_agents.Version,
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

func GetCustomAgentMetrics(c *gin.Context, db *gorm.DB) (models.CustomIntegrationsMetrics, error) {
	metrics, err := new(models.OrganisationIntegrations).GetCustomAgentCountMetrics(db)
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}

func GetCustomAgentByID(c *gin.Context, db *gorm.DB, agent_id string) (models.AdminAgentResp, error) {
	agent, err := new(models.OrganisationIntegrations).GetCustomAgentByID(db, agent_id)
	if err != nil {
		return agent, err
	}

	return agent, nil
}

func AdminDeleteCustomAgentApp(db *gorm.DB, logger utility.Logger, agentID string) (error, int) {
	var org_agent models.OrganisationIntegrations

	err, code := org_agent.AdminDeleteCustomAgentApp(db, logger, agentID)
	if err != nil {
		return err, code
	}

	return nil, code
}
