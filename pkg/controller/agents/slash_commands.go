package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AddAgentSlashCommand(c *gin.Context) {
	var (
		req models.AddSlashCommandRequest
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

	response, err := agents.AddAgentsSlashCommand(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to add agent slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to add agent slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent setting added successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent setting added successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAgentSlashCommands(c *gin.Context) {
	var (
		org_id   = c.Param("org_id")
		agent_id = c.Param("agent_id")
	)

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

	response, err := agents.GetIntegrationSlashCommands(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to get agent slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get agent slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent slashcommands retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent slashcommands retrieved successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateAgentSlashCommand(c *gin.Context) {
	var (
		req        models.UpdateSlashCommandRequest
		org_id     = c.Param("org_id")
		agent_id   = c.Param("agent_id")
		command_id = c.Param("command_id")
	)

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

	if _, err := uuid.Parse(command_id); err != nil {
		base.Logger.Error("invalid command id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid command id format", "failed to decode agent id", nil)
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
		"org_id":     org_id,
		"agent_id":   agent_id,
		"command_id": command_id,
	}

	response, err := agents.UpdateAgentSlashCommand(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to update agent slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update agent slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent setting updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent setting updated successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteAgentSlashCommand(c *gin.Context) {
	var (
		org_id     = c.Param("org_id")
		agent_id   = c.Param("agent_id")
		command_id = c.Param("command_id")
	)

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

	if _, err := uuid.Parse(command_id); err != nil {
		base.Logger.Error("invalid command id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid command id format", "failed to decode command id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":     org_id,
		"agent_id":   agent_id,
		"command_id": command_id,
	}

	err := agents.DeleteAgentSlashCommand(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to delete agent slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to delete agent slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent slashcommands deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent slashcommands deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAllOrgSlashCommands(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, err := agents.GetAllOrgSlashCommands(base.Db.Postgresql, org_id)
	if err != nil {
		base.Logger.Error("Failed to get all organisation slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get all organisation slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Organisation slashcommands retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation slashcommands retrieved successfully", response)
	c.JSON(http.StatusOK, rd)
}
