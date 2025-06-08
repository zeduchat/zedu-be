package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetAllAgentApp(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	agents, err := agents.GetAllAgentApp(c, org_id, base.Db.Postgresql)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			base.Logger.Error("agents not found", err)
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "agents not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			base.Logger.Error("Failed to fetch agents", err)
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch agents", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents retrieved successfully.", agents)
	c.JSON(http.StatusOK, rd)
}

// Fetch Custom Agents with pagination
func (base *Controller) GetCustomAgentApp(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	agents, paginationResponse, err, code := agents.GetCustomAgentApp(c, org_id, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch agents", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch agents", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]interface{}{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(agents),
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents retrieved successfully.", agents, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateAgentApp(c *gin.Context) {
	var req models.UpdateAgent
	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Input validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Input validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	updatedAgents, err := agents.UpdateAgentApp(req, ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to update agent app", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to update agent app", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Agents updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agents updated successfully", updatedAgents)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteAgentApp(c *gin.Context) {
	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	err := agents.DeleteAgentApp(ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to delete agent app", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to delete agent app", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Agent app deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent app deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

// Delete Custom agent
func (base *Controller) DeleteCustomAgentApp(c *gin.Context) {
	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		OrganisationID: org_id,
		AgentID:        agent_id,
	}

	err, code := agents.DeleteCustomAgentApp(base.Db.Postgresql, *base.Logger, ids)
	if err != nil {
		base.Logger.Error("Failed to delete agent app", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to delete agent app", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Agent app deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent app deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ChangeAgentStatus(c *gin.Context) {
	org_id := c.Param("org_id")
	var req models.ChangeAgentStatus

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.AgentID); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": req.AgentID,
	}

	err := agents.ChangeStatus(ids, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to set agent app status", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to set agent app status", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent app status set successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent app status set successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ChangeOrgChannelIntSendBackStatus(c *gin.Context) {
	org_id := c.Param("org_id")
	channel_id := c.Param("channel_id")
	var req models.ChangeAgentStatus

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel_id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.AgentID); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":     org_id,
		"channel_id": channel_id,
		"agent_id":   req.AgentID,
	}

	err := agents.ChangeAgentSendBackStatus(ids, req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to set agent app status", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to set agent app status", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent app status set successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent app status set successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateJSONSchema(c *gin.Context) {
	var (
		req models.UpdateJSONSchemaRequest
	)

	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	err := agents.UpdateJSONSchema(ids, req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to update JSON schema", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to update JSON schema", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "JSON schema updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateCustomAgent(c *gin.Context) {
	var (
		req models.CustomIntegrationRequest
	)

	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	err := agents.UpdateCustomAgent(ids, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to update JSON schema", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to update custom agent", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Custom agent updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetActivatedOrganizations(c *gin.Context) {
	agent_id := c.Param("agent_id")
	api_key := c.Query("api_key")

	if api_key == "" {
		base.Logger.Error("missing api_key")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "missing api_key", "api_key is required", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	organisations, err, code := agents.GetActivatedOrganizations(base.Db.Postgresql, agent_id, api_key)

	if err != nil {
		base.Logger.Error("Failed to fetch organisation that has activated this agent!!", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Successful", organisations)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAgentSettingsAllOrgs(c *gin.Context) {
	agent_id := c.Param("agent_id")
	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	settings, err := agents.GetAgentSettingsAllOrgs(base.Db.Postgresql, agent_id)
	if err != nil {
		base.Logger.Error("Failed to get agent settings across all organisations", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to get agent settings across all organisations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Agent settings across all organisations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent settings across all organisations retrieved successfully.", settings)
	c.JSON(http.StatusOK, rd)
}

// Create custom agent
func (base *Controller) CreateCustomAgent(c *gin.Context) {
	var (
		req models.CustomIntegrationRequest
	)

	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, err := agents.CreateCustomAgent(org_id, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to Create Custom Agent, invalid url:  "+req.JSONUrl, err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to create agent", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Custom agent created successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent created successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

// Agent Settings

func (base *Controller) UpdateCustomAgentSettings(c *gin.Context) {
	var (
		req models.CustomIntegrationSettingRequest
	)

	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	err := agents.UpdateCustomAgentSettings(ids, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to update custom agent settings", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to update custom agent settings", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Custom agent settings updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetCustomAgentStatus(c *gin.Context) {
	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	integration_setting, code, err := agents.GetCustomAgentStatus(ids, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch custom agents settings", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch custom agents settings", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents setting retrieved successfully.", integration_setting)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetCustomAgentSettings(c *gin.Context) {

	org_id := c.Param("org_id")
	agent_id := c.Param("agent_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	integration_setting, code, err := agents.GetCustomAgentSettings(ids, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch custom agents settings", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch custom agents settings", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents setting retrieved successfully.", integration_setting)
	c.JSON(http.StatusOK, rd)
}

// Agents External
func (base *Controller) GetCustomAgentSettingsExteranl(c *gin.Context) {
	api_key := c.Query("api_key")

	if api_key == "" {
		base.Logger.Error("made a request without api_key")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "api_key is missing in query params, consult the docs", "failed to parse api_key", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	key := config.Config.Server.EncKey

	porg_id, pint_id, err := utility.ValidateExternalApiKey(api_key, key)

	if err != nil {
		base.Logger.Error("An error occured: %s", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to parse api_key", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"porg_id":   porg_id,
		"pagent_id": pint_id,
	}

	integration_setting, code, err := agents.GetCustomAgentSettingsExteranl(ids, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch custom agents settings", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch agents settings", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents setting retrieved successfully.", integration_setting)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateCustomAgentSettingsExternal(c *gin.Context) {
	var (
		req models.CustomIntegrationSettingRequest
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	auth_credentials, ok := req.SettingEntry["auth_credentials"].(map[string]interface{})

	if !ok {
		base.Logger.Error("auth_credentials is missing in request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "auth_credentials is missing in request body, consult telex docs", "invalid auth_credentials supplied", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	_, ok = req.SettingEntry["settings"].([]interface{})
	if !ok {
		base.Logger.Error("settings is missing in request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "settings field not returned, consult telex docs", "invalid request body supplied", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	api_key, ok := auth_credentials["telex_api_key"].(string)

	if !ok {
		base.Logger.Error("api_key is missing in request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "api_key is missing in request body", "invalid api_key supplied", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	key := config.Config.Server.EncKey

	porg_id, pint_id, err := utility.ValidateExternalApiKey(api_key, key)

	if err != nil {
		base.Logger.Error("An error occured: %s", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to parse api_key", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"porg_id":       porg_id,
		"pagent_id":     pint_id,
		"telex_api_key": api_key,
	}

	err = agents.UpdateCustomAgentSettingsExternal(ids, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to update agent settings", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to update agent settings", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent settings updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) AgentCallback(c *gin.Context) {

	api_key := c.GetHeader("X-TELEX-API-KEY")

	if api_key == "" {
		base.Logger.Error("X-TELEX-API-KEY is missing in request header")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "X-TELEX-API-KEY is missing in request header", "invalid api key supplied", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	key := config.Config.Server.EncKey

	porg_id, pint_id, err := utility.ValidateExternalApiKey(api_key, key)

	if err != nil {
		base.Logger.Error("An error occured: %s", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to parse api key", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"porg_id":       porg_id,
		"pagent_id":     pint_id,
		"telex_api_key": api_key,
	}

	err = agents.AgentCallback(ids, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to process agent callback", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to process agent callback", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent callback received successfully", nil)
	c.JSON(http.StatusOK, rd)
}
