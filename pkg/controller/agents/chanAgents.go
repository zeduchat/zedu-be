package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetOrganisationChannelAgents(c *gin.Context) {
	channel_id := c.Param("channel_id")
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to decode channel id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	agents, paginationResponse, code, err := agents.GetOrganisationChannelAgents(base.Db.Postgresql, channel_id, org_id, c, base.ExtReq)

	if err != nil {
		base.Logger.Error("Failed to get channel agents")
		rd := utility.BuildErrorResponse(code, "error", "Failed to get channel agents", err, nil)
		c.JSON(code, rd)
		return
	}


	base.Logger.Info("Channel agents retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "Channel agents retrieved successfully", agents, paginationResponse)
	c.JSON(code, rd)
}

func (base *Controller) ActivateDeactivateChannelAgent(c *gin.Context) {
	org_id := c.Param("org_id")
	channel_id := c.Param("channel_id")
	agent_id := c.Param("agent_id")
	req := models.ActivateChannelAgent{}

	var msg string

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to decode channel id", nil)
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
		base.Logger.Error("Failed to bind request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to bind request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"channel_id":      channel_id,
		"agent_id":  agent_id,
	}

	err := agents.ActivateChannelAgent(ids, req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to activate channel agent")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if req.Status {
		msg = "Channel agent activated successfully"
	} else {
		msg = "Channel agent deactivated successfully"
	}

	base.Logger.Info(msg)
	rd := utility.BuildSuccessResponse(http.StatusOK, msg, nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) AgentChannels(c *gin.Context) {
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
		"organisation_id": org_id,
		"agent_id":  agent_id,
	}

	res, err := agents.AgentChannels(ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to activate channel agent", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent channels fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent channels fetched successfully", res)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CheckAgentIsActive(c *gin.Context) {
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
		"organisation_id": org_id,
		"agent_id":  agent_id,
	}

	res, err := agents.CheckAgentIsActive(ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to fetch agent status", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Agent status fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent status fetched successfully", res)
	c.JSON(http.StatusOK, rd)
}
