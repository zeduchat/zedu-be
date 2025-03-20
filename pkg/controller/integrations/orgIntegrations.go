package integrations

import (
	"fmt"
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

func (base *Controller) GetAllIntegrationApp(c *gin.Context) {
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

	base.Logger.Info("integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "integrations retrieved successfully.", agents)
	c.JSON(http.StatusOK, rd)
}

// Fetch Custom Integrations with pagination
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
		fmt.Println(err)
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

	ids := map[string]string{
		"org_id":   org_id,
		"agent_id": agent_id,
	}

	err, code := agents.DeleteCustomAgentApp(ids, base.Db.Postgresql)
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

	err := agents.ChangeAgentStatus(ids, req, base.Db.Postgresql, base.ExtReq)
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
	var req models.ChangeIntegrationStatus

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

	if _, err := uuid.Parse(req.IntegrationID); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"channel_id":     channel_id,
		"integration_id": req.IntegrationID,
	}

	err := integrations.ChangeIntegrationSendBackStatus(ids, req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to set integration app status", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to set integration app status", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration app status set successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration app status set successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateJSONSchema(c *gin.Context) {
	var (
		req models.UpdateJSONSchemaRequest
	)

	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
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
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	err := integrations.UpdateJSONSchema(ids, req, base.Db.Postgresql)
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

func (base *Controller) UpdateCustomIntegration(c *gin.Context) {
	var (
		req models.CustomIntegrationRequest
	)

	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
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
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	err := integrations.UpdateCustomIntegration(ids, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to update JSON schema", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to update custom integration", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Custom integration updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetIntegrationSettingsAllOrgs(c *gin.Context) {
	integration_id := c.Param("integration_id")
	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	settings, err := integrations.GetIntegrationSettingsAllOrgs(base.Db.Postgresql, integration_id)
	if err != nil {
		base.Logger.Error("Failed to get integration settings across all organisations", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to get integration settings across all organisations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Integration settings across all organisations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration settings across all organisations retrieved successfully.", settings)
	c.JSON(http.StatusOK, rd)
}

// Create custom integration
func (base *Controller) CreateCustomIntegration(c *gin.Context) {
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

	err := integrations.CreateCustomIntegration(org_id, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to Create Custom Integration, invalid url:  "+req.JSONUrl, err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to create custom integration", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Custom integration created successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Custom integration created successfully", nil)
	c.JSON(http.StatusCreated, rd)
}

// Integration Settings

func (base *Controller) UpdateCustomIntegrationSettings(c *gin.Context) {
	var (
		req models.CustomIntegrationSettingRequest
	)

	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
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
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	err := integrations.UpdateCustomIntegrationSettings(ids, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to update custom integration settings", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to update custom integration settings", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Custom integration settings updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetCustomIntegrationStatus(c *gin.Context) {
	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	integration_setting, code, err := integrations.GetCustomIntegrationStatus(ids, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		fmt.Println(err)
		base.Logger.Error("Failed to fetch custom integrations settings", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch custom integrations settings", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "integrations setting retrieved successfully.", integration_setting)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetCustomIntegrationSettings(c *gin.Context) {

	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	integration_setting, code, err := integrations.GetCustomIntegrationSettings(ids, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		fmt.Println(err)
		base.Logger.Error("Failed to fetch custom integrations settings", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch custom integrations settings", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "integrations setting retrieved successfully.", integration_setting)
	c.JSON(http.StatusOK, rd)
}

// Integrations External
func (base *Controller) GetCustomIntegrationSettingsExteranl(c *gin.Context) {
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
		"porg_id":         porg_id,
		"pintegration_id": pint_id,
	}

	integration_setting, code, err := integrations.GetCustomIntegrationSettingsExteranl(ids, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch custom integrations settings", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch integrations settings", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "integrations setting retrieved successfully.", integration_setting)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateCustomIntegrationSettingsExternal(c *gin.Context) {
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
		"porg_id":         porg_id,
		"pintegration_id": pint_id,
		"telex_api_key":   api_key,
	}

	err = integrations.UpdateCustomIntegrationSettingsExternal(ids, req, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to update integration settings", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to update integration settings", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("JSON schema updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration settings updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}
