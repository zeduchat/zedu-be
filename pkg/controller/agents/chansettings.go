package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AddChannelAgentSetting(c *gin.Context) {
	var (
		req        models.AddIntegrationSettingsRequest
		org_id     = c.Param("org_id")
		agent_id   = c.Param("agent_id")
		channel_id = c.Param("channel_id")
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
	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to decode channel id", nil)
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
		"channel_id": channel_id,
	}

	err := agents.AddChannelAgentSettings(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to add channel agent settings", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to add channel agent setting", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Channel Agent setting added successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel Agent setting added successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetChannelAgentSetting(c *gin.Context) {
	var (
		org_id     = c.Param("org_id")
		agent_id   = c.Param("agent_id")
		channel_id = c.Param("channel_id")
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
	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel agent id format", "failed to decode channel agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":     org_id,
		"agent_id":   agent_id,
		"channel_id": channel_id,
	}

	setting, err := agents.GetChannelAgentSettings(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to get channel agent settings", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get channel agent settings", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Channel Agent settings retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel Agent settings retrieved successfully", setting)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateAgentSetting(c *gin.Context) {
	var (
		req        models.UpdateIntegrationSettingsRequest
		org_id     = c.Param("org_id")
		agent_id   = c.Param("agent_id")
		channel_id = c.Param("channel_id")
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

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel agent id format", "failed to decode channel agent id", nil)
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
		"channel_id": channel_id,
	}

	err := agents.UpdateChannelAgentSettings(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to update channel agent setting", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update channel agent setting", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Channel Agent setting updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel Agent setting updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteChannelAgentSetting(c *gin.Context) {
	var (
		org_id     = c.Param("org_id")
		agent_id   = c.Param("agent_id")
		channel_id = c.Param("channel_id")
		setting_id = c.Param("setting_id")
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
	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel agent id format", "failed to decode channel agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(setting_id); err != nil {
		base.Logger.Error("invalid setting id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid setting id format", "failed to decode setting id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":     org_id,
		"agent_id":   agent_id,
		"channel_id": channel_id,
		"setting_id": setting_id,
	}

	err := agents.DeleteChannelAgentSettings(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to delete channel agent setting", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to delete channel agent setting", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Channel Agent setting deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel Agent setting deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}
