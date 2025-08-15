package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreateAgentSkill(c *gin.Context) {
	var req models.CreateAgentSkillRequest

	agentID := c.Param("agents_id")

	if _, err := uuid.Parse(agentID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid request body", err, nil)
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

	req.AgentId = agentID

	resp, code, err := agents.CreateAgentSkill(req, base.Db.Postgresql, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), "failed to create agent skill", nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Custom agent skill added successfully")
	c.JSON(code, utility.BuildSuccessResponse(code, "Agent skill created", resp))
}

func (base *Controller) GetAgentSkill(c *gin.Context) {
	agent_id := c.Param("agents_id")

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	skills, pagination, err, code := agents.GetAgentSkills(agent_id, base.Db.Postgresql, c)
	if err != nil {
		c.JSON(code, utility.BuildErrorResponse(code, "error", err.Error(), "failed to get agent skills", nil))
		return
	}
	c.JSON(code, utility.BuildSuccessResponse(code, "Agent skills retrieved", skills, pagination))
}

func (base *Controller) UpdateAgentSkill(c *gin.Context) {
	skill_id := c.Param("integration_id")
	var updateData map[string]interface{}

	if _, err := uuid.Parse(skill_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integraion id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid request body", err, nil))
		return
	}
	updated, err := agents.UpdateAgentSkill(skill_id, updateData, base.Db.Postgresql)
	if err != nil {
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to update agent skill", nil))
		return
	}
	c.JSON(http.StatusOK, utility.BuildSuccessResponse(http.StatusOK, "Agent skill updated", updated))
}

func (base *Controller) DeleteAgentSkill(c *gin.Context) {
	skillID := c.Param("integration_id")
	if err := agents.DeleteAgentSkill(skillID, base.Db.Postgresql); err != nil {
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to delete agent skill", nil))
		return
	}
	c.JSON(http.StatusOK, utility.BuildSuccessResponse(http.StatusOK, "Agent skill deleted", nil))
}
