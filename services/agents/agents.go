package agents

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
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

	resp, paginationResult, err, code := org_agents.GetCustomAgentApps(db, org_id, c)

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
					IsActive:       false,
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

			response_data := response.(map[string]any)

			data_r := response_data["data"].(map[string]any)

			description := data_r["descriptions"].(map[string]any)
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
			continue
		}

		agent := models.Integrations{
			ID:             org_agents.IntegrationID,
			Name:           org_agents.AppName,
			JSONUrl:        org_agents.JSONUrl,
			AppUrl:         org_agents.AppUrl,
			AppLogo:        org_agents.AppLogo,
			AppDescription: org_agents.AppDescription,
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

func GetSystemAgentApps(c *gin.Context, db *gorm.DB, extReq request.ExternalRequest) (*[]models.AgentResp, postgresql.PaginationResponse, error, int) {
	var (
		agents  models.Integrations
		botResp []models.AgentResp = make([]models.AgentResp, 0)
	)

	resp, paginationResult, err, code := agents.GetSystemAgentApps(db, c, "")

	if err != nil {
		return &[]models.AgentResp{}, postgresql.PaginationResponse{}, err, code
	}

	for _, agents := range resp {

		parts := strings.Split(agents.ID, "-")
		lastPart := parts[len(parts)-1]

		agent := models.AgentResp{
			ID:          agents.ID,
			Name:        agents.Name,
			Title:       agents.Title,
			Tone:        agents.Tone,
			Visibility:  agents.Visibility,
			Avatar:      agents.AppLogo,
			Description: agents.AppDescription,
			IsActive:    agents.IsActive,
			Category:    agents.Category,
			Stars:       agents.Stars,
			AgentSlug:   fmt.Sprintf("%s-%s", slug.Make(agents.Name), lastPart),
		}

		botResp = append(botResp, agent)

	}

	return &botResp, paginationResult, nil, code
}

func GetSystemAgentApp(c *gin.Context, db *gorm.DB, int_id string, extReq request.ExternalRequest) (*models.AgentResp, error, int) {
	var agents models.Integrations

	agent, err, code := agents.GetSystemAgentApp(db, int_id, c)

	if err != nil {
		return &models.AgentResp{}, err, code
	}

	resp := models.AgentResp{

		ID:          agent.ID,
		Name:        agent.Name,
		Title:       agent.Title,
		Tone:        agent.Tone,
		Visibility:  agent.Visibility,
		Avatar:      agent.AppLogo,
		Description: agent.AppDescription,
		IsActive:    agent.IsActive,
		Category:    agent.Category,
		Stars:       agent.Stars,
		Snapshot:    agent.Snapshot,
		HowItWorks:  agent.HowItWorks,
		Benefits:    agent.Benefits,
		WhyUse:      agent.WhyUse,
	}
	return &resp, nil, code
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
		deserialize_settings map[string]any
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

	auth_credentials, ok := deserialize_settings["auth_credentials"].(map[string]any)

	if ok {
		api_key, _ = auth_credentials["agent_api_key"].(string)
	}

	// send api key to agent

	parsedURL, _ := url.Parse(orgIntegration.JSONUrl)
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	dataPayload := map[string]any{
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

func CreateCustomAgent(req models.CreateAgentRequest, db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger) (models.AgentResp, int, error) {
	var (
		orgIntegration models.OrganisationIntegrations
		organisation   models.Organisation
		agentSettings  models.CustomIntegrationsSetting
		int_resp       models.AgentResp
		org            models.Organisation
	)

	isMember, err := org.CheckUserIsMemberOfOrg(req.UserId, req.OrgId, db)
	if err != nil {
		return int_resp, http.StatusBadRequest, err
	}
	if !isMember {
		return int_resp, http.StatusBadRequest, errors.New("user not a member of organisation")
	}

	organisationExists := postgresql.CheckExists(db, &organisation, "id = ?", req.OrgId)
	if !organisationExists {
		return int_resp, http.StatusNotFound, errors.New("organisation does not exist")
	}

	psk, err := models.GenerateAgentKey()
	if err != nil {
		return int_resp, http.StatusInternalServerError, err
	}

	orgIntegration.OrgID = req.OrgId
	orgIntegration.IsActive = true
	orgIntegration.IsSystem = false
	orgIntegration.ID = utility.GenerateUUID()
	orgIntegration.AppName = req.Name
	orgIntegration.Title = req.Title
	orgIntegration.Tone = req.Tone
	orgIntegration.Visibility = req.Visibility
	orgIntegration.AppDescription = req.Description
	orgIntegration.OwnerID = req.UserId
	orgIntegration.IntegrationID = utility.GenerateUUID()
	orgIntegration.SystemPrompts = []models.SystemPrompts{
		{
			Name:    "System Prompt",
			Content: fmt.Sprintf("Your name is %s and your job is %s", req.Name, req.Description),
			Type:    "system_prompt",
			Id:      utility.GenerateUUID(),
		},
	}
	orgIntegration.AppLogo = req.Avatar
	orgIntegration.PreSharedKey = psk
	err = orgIntegration.CreateOrganisationIntegration(db)
	if err != nil {
		return int_resp, http.StatusInternalServerError, err
	}
	auth_credentials := map[string]any{"agent_auth_credentials": "Not-Set-Yet"}
	settings_data := map[string]any{"settings": ""}
	auth_credentials["agent_api_key"] = psk
	settings_data["auth_credentials"] = auth_credentials
	settingJsonData, err := json.Marshal(settings_data)
	if err != nil {
		return int_resp, http.StatusInternalServerError, fmt.Errorf("error serializing to JSON: %v", err)
	}
	serialized_settings := string(settingJsonData)
	agentSettings.SettingEntry = serialized_settings
	agentSettings.OrgID = req.OrgId
	agentSettings.IntegrationID = orgIntegration.IntegrationID
	agentSettings.IsSystem = false
	agentSettings.ID = utility.GenerateUUID()
	err = agentSettings.CreateIntegrationSettings(db)
	if err != nil {
		return int_resp, http.StatusInternalServerError, err
	}

	int_resp = models.AgentResp{
		ID:          orgIntegration.IntegrationID,
		Name:        orgIntegration.AppName,
		Title:       orgIntegration.Title,
		Tone:        orgIntegration.Tone,
		Visibility:  orgIntegration.Visibility,
		Avatar:      orgIntegration.AppLogo,
		Description: orgIntegration.AppDescription,
		IsActive:    orgIntegration.IsActive,
	}

	return int_resp, http.StatusCreated, nil
}

func validateJSONURL(jsonUrl string) error {
	validator := utility.NewURLValidator()
	if err := validator.Validate(jsonUrl); err != nil {
		return fmt.Errorf("invalid JSON URL: %v", err)
	}
	return nil
}

// Update CustomIntegration
func UpdateCustomAgent(req models.CreateAgentRequest, db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger) (int, error) {

	var orgIntegration models.OrganisationIntegrations

	exists := postgresql.CheckExists(db, &orgIntegration, "org_id = ? AND integration_id = ?", req.OrgId, req.AgentId)
	if !exists {
		return http.StatusNotFound, errors.New("organisation does not have that agent")
	}

	code, err := orgIntegration.UpdateCustomAgent(db, req)

	if err != nil {
		return code, err
	}

	return code, nil
}

func UpdateCustomAgentPrompt(req models.UpdateAgentPromptRequest, db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger) (int, error) {

	var orgIntegration models.OrganisationIntegrations

	code, err := orgIntegration.UpdateCustomAgentPrompt(db, req)

	if err != nil {
		return code, err
	}

	return code, nil
}

func CreateCustomAgentPrompt(req models.UpdateAgentPromptRequest, db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger) (int, error) {

	var orgIntegration models.OrganisationIntegrations

	code, err := orgIntegration.CreateCustomAgentPrompt(db, req)

	if err != nil {
		return code, err
	}

	return code, nil
}

func DeleteCustomAgentPrompt(req models.UpdateAgentPromptRequest, db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger) (int, error) {

	var orgIntegration models.OrganisationIntegrations

	code, err := orgIntegration.DeleteCustomAgentPrompt(db, req)

	if err != nil {
		return code, err
	}

	return code, nil
}

// Update CustomIntegration
func FetchCustomAgent(req models.CreateAgentRequest, db *gorm.DB, extReq request.ExternalRequest, logger *utility.Logger) (*models.AgentResp, int, error) {

	var org_agents models.OrganisationIntegrations
	var agents models.Integrations

	exists := postgresql.CheckExists(db, &org_agents, "org_id = ? AND integration_id = ?", req.OrgId, req.AgentId)
	if !exists {
		exists = postgresql.CheckExists(db, &agents, "id = ?", req.AgentId)

		if !exists {
			return &models.AgentResp{}, http.StatusNotFound, errors.New("Agent does not exists")
		}

		parts := strings.Split(agents.ID, "-")
		lastPart := parts[len(parts)-1]

		resp := models.AgentResp{
			ID:            agents.ID,
			Name:          agents.Name,
			Title:         agents.Title,
			Tone:          agents.Tone,
			Visibility:    agents.Visibility,
			Avatar:        agents.AppLogo,
			Description:   agents.AppDescription,
			IsActive:      false,
			SystemPrompts: agents.SystemPrompts,
			Snapshot:      agents.Snapshot,
			HowItWorks:    agents.HowItWorks,
			Benefits:      agents.Benefits,
			WhyUse:        agents.WhyUse,
			AgentSlug:     fmt.Sprintf("%s-%s", slug.Make(agents.Name), lastPart),
		}
		return &resp, http.StatusOK, nil
	}

	parts := strings.Split(org_agents.IntegrationID, "-")
	lastPart := parts[len(parts)-1]

	resp := models.AgentResp{
		ID:            org_agents.IntegrationID,
		Name:          org_agents.AppName,
		Title:         org_agents.Title,
		Tone:          org_agents.Tone,
		Visibility:    org_agents.Visibility,
		Avatar:        org_agents.AppLogo,
		Description:   org_agents.AppDescription,
		IsActive:      org_agents.IsActive,
		SystemPrompts: org_agents.SystemPrompts,
		AgentSlug:     fmt.Sprintf("%s-%s", slug.Make(agents.Name), lastPart),
	}

	if exists = postgresql.CheckExists(db, &agents, "id = ?", req.AgentId); exists {
		resp.Benefits = agents.Benefits
		resp.Snapshot = agents.Snapshot
		resp.WhyUse = agents.WhyUse
		resp.HowItWorks = agents.HowItWorks
	}

	return &resp, http.StatusOK, nil
}

func GetOrganisationChannelAgents(db *gorm.DB, channel_id, org_id string, c *gin.Context, extReq request.ExternalRequest) ([]models.AgentResp, postgresql.PaginationResponse, int, error) {
	var (
		ocIntegrations models.OrganisationChannelsIntegrations
		agentResp      []models.AgentResp = make([]models.AgentResp, 0)
	)

	agents, paginationResponse, code, err := ocIntegrations.GetOrganisationChannelAgents(db, channel_id, org_id, c)

	if err != nil {
		return nil, paginationResponse, code, err
	}

	for _, org_agents := range agents {

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

		agentResp = append(agentResp, agent)

	}

	return agentResp, paginationResponse, code, nil
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

func GetCustomAgentSettings(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]any, int, error) {

	var (
		orgIntegration models.OrganisationIntegrations
		ucis           models.CustomIntegrationsSetting

		deserialize_settings map[string]any
		resp                 map[string]any
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

	resp = make(map[string]any)

	resp["is_system"] = ucis.IsSystem
	resp["is_active"] = orgIntegration.IsActive
	resp["settings"] = deserialize_settings["settings"]

	return resp, http.StatusOK, nil
}

func GetCustomAgentStatus(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]any, int, error) {

	var (
		orgIntegration       models.OrganisationIntegrations
		ucis                 models.CustomIntegrationsSetting
		deserialize_settings map[string]any
	)
	status := make(map[string]any)

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

	auth_credentials, ok := deserialize_settings["auth_credentials"].(map[string]any)

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

func GetCustomAgentSettingsExteranl(ids map[string]string, db *gorm.DB, extReq request.ExternalRequest) (map[string]any, int, error) {
	var (
		ucis                 models.CustomIntegrationsSetting
		deserialize_settings map[string]any
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
		deserialize_settings map[string]any
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
	auth_credentials, ok := deserialize_settings["auth_credentials"].(map[string]any)
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

func GetAllCustomAgent(c *gin.Context, db *gorm.DB) ([]models.PartialOrganisationIntegration, postgresql.PaginationResponse, error, int) {
	search := c.Query("search")
	isSystem := c.DefaultQuery("is_system", "true")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	is_active := c.DefaultQuery("is_active", "true")

	isSystemBool, err := strconv.ParseBool(isSystem)
	if err != nil {
		isSystemBool = false
	}

	active, err := strconv.ParseBool(is_active)
	if err != nil {
		active = false
	}

	var (
		resp             []models.PartialOrganisationIntegration
		paginationResult postgresql.PaginationResponse
		code             int
		fetchErr         error
		orgAgentsModel   models.PartialOrganisationIntegration
	)

	if isSystemBool {
		resp, paginationResult, fetchErr, code = orgAgentsModel.GetAllSystemAgent(db, c, search, sortBy, sortOrder, active)
	} else {
		resp, paginationResult, fetchErr, code = orgAgentsModel.GetAllCustomAgent(db, c, search, sortBy, sortOrder, active)
	}

	if fetchErr != nil {
		return nil, postgresql.PaginationResponse{}, fetchErr, code
	}

	return resp, paginationResult, nil, code
}

func GetCustomAgentMetrics(c *gin.Context, db *gorm.DB) (models.CustomIntegrationsMetrics, error) {
	metrics, err := new(models.OrganisationIntegrations).GetCustomAgentCountMetrics(db)
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}

func GetCustomAgentByID(c *gin.Context, db *gorm.DB, agentID string, source string) (any, error) {
	var (
		agent any
		err   error
	)

	if source == "organization" {
		agent, err = new(models.OrganisationIntegrations).GetCustomAgentByID(db, agentID)
	} else {
		agent, err = new(models.Integrations).GetAgentByID(db, agentID)
	}

	if err != nil {
		return models.AdminAgentResp{}, err
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

func AdminUpdateAgent(req models.AdminUpdateAgent, agent_id string, db *gorm.DB) (models.OrganisationIntegrations, error) {
	var agent models.OrganisationIntegrations

	updatedAgent, err := agent.AdminUpdateAgent(db, agent_id, req)
	if err != nil {
		return models.OrganisationIntegrations{}, err
	}

	return updatedAgent, nil
}

func CreateSystemAgent(req models.CustomIntegrationRequest, db *gorm.DB, extReq request.ExternalRequest, admin_id string) (models.AgentResp, error) {
	var (
		systemIntegration models.Integrations
		int_resp          models.AgentResp
		agentSettings     models.IntegrationSettings
	)

	err := validateJSONURL(req.JSONUrl)
	if err != nil {
		return int_resp, err
	}

	agentID, err := utility.GenerateUUIDFromString(req.JSONUrl)
	if err != nil {
		return int_resp, fmt.Errorf("error generating agent ID from JSON URL: %v", err)
	}

	exists := postgresql.CheckExists(db, &systemIntegration, "id = ?", agentID)
	if exists {
		return int_resp, errors.New("organisation already has that agent")
	}

	data := map[string]string{"url": req.JSONUrl}
	response, err := extReq.SendExternalRequest(request.AgentJsonContent, data)

	if err != nil {
		return int_resp, errors.New("failed to create agent, invalid JSON supplied")
	}

	data_r, ok := response.(map[string]any)
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

	systemIntegration.JSONUrl = req.JSONUrl
	systemIntegration.ID = agentID
	systemIntegration.IsActive = true
	systemIntegration.IsSystem = true
	systemIntegration.ID = utility.GenerateUUID()
	systemIntegration.Name = data_r["name"].(string)
	systemIntegration.AppDescription = data_r["description"].(string)
	systemIntegration.AppUrl = data_r["url"].(string)
	systemIntegration.Prices = payload.Prices
	systemIntegration.Provider = payload.Provider
	systemIntegration.Version = payload.Version
	systemIntegration.DefaultInputModes = payload.DefaultInputModes
	systemIntegration.DefaultOutputModes = payload.DefaultOutputModes
	systemIntegration.Skills = payload.Skills
	systemIntegration.IsPaid = payload.IsPaid
	systemIntegration.PreSharedKey = psk
	systemIntegration.OwnerID = admin_id

	err = systemIntegration.CreateSystemIntegration(db)
	if err != nil {
		return int_resp, err
	}

	settings := ""

	settings_data := map[string]any{"settings": settings}

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
	agentSettings.IsSystem = true
	agentSettings.OrgID = nil
	agentSettings.IntegrationID = systemIntegration.ID
	agentSettings.FormFieldValue = serialized_settings
	agentSettings.FormFieldLabel = "Agent Auth"

	err = agentSettings.CreateSystemIntegrationSettings(db)
	if err != nil {
		return int_resp, errors.New("failed to create agent settings")
	}

	int_resp = models.AgentResp{
		ID:          systemIntegration.ID,
		Name:        systemIntegration.Name,
		Avatar:      systemIntegration.AppLogo,
		Description: systemIntegration.AppDescription,
		IsActive:    systemIntegration.IsActive,
	}

	return int_resp, nil
}

func AdminDeleteSystemAgentApp(db *gorm.DB, logger utility.Logger, agentID string) (error, int) {
	var agent models.Integrations

	err, code := agent.AdminDeleteSystemAgentApp(db, logger, agentID)
	if err != nil {
		return err, code
	}

	return nil, code
}

func GetAgentBills(c *gin.Context, db *gorm.DB) ([]models.IntegrationBillsResponse, postgresql.PaginationResponse, error, int) {
	var (
		resp             []models.IntegrationBillsResponse
		paginationResult postgresql.PaginationResponse
		code             int
		fetchErr         error
		agentBills       models.IntegrationBillsResponse
	)

	resp, paginationResult, fetchErr, code = agentBills.GetAgentBills(db, c)

	if fetchErr != nil {
		return nil, postgresql.PaginationResponse{}, fetchErr, code
	}

	return resp, paginationResult, nil, code
}

func GetOrgAgentBills(c *gin.Context, db *gorm.DB, org_id string) ([]models.IntegrationBillsResponse, postgresql.PaginationResponse, error, int) {
	var (
		resp             []models.IntegrationBillsResponse
		paginationResult postgresql.PaginationResponse
		code             int
		fetchErr         error
		agentBills       models.IntegrationBillsResponse
	)

	resp, paginationResult, fetchErr, code = agentBills.GetOrgAgentBills(db, c, org_id)

	if fetchErr != nil {
		return nil, postgresql.PaginationResponse{}, fetchErr, code
	}

	return resp, paginationResult, nil, code
}

func UploadAgentAvatar(logger *utility.Logger, uniqueId string, file []byte, ext string) (string, error) {
	if file != nil {
		logoId := strings.Split(uniqueId, "-")[4]
		filename := fmt.Sprintf("agent_logo_%s%s", logoId, ext)

		picUrl, err := minio.UploadProfilePic(logger, filename, bytes.NewReader(file), int64(len(file)))
		if err != nil {
			return "", err
		}
		return picUrl, nil
	}
	return "", nil
}

func PublishAgent(req models.PublishAgentRequest, db *gorm.DB) (int, error) {
	var agent models.OrganisationIntegrations

	code, err := agent.PublishAgent(req, db)
	if err != nil {
		return code, err
	}

	return code, nil
}
