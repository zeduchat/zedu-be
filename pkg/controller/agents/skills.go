package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
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

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)
	req.UserId = userClaims["user_id"].(string)

	resp, code, err := agents.CreateAgentSkill(req, base.Db.Postgresql, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to create agent skill, err: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), "failed to create agent skill", nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Custom agent skill added successfully")
	c.JSON(code, utility.BuildSuccessResponse(code, "Agent skill created", resp))
}

func (base *Controller) GetAgentSkills(c *gin.Context) {
	var req models.CreateAgentSkillRequest
	agent_id := c.Param("agents_id")

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)

	if _, err := uuid.Parse(req.OrgId); err != nil || req.OrgId == "" {
		base.Logger.Info("invalid organization id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	skills, pagination, err, code := agents.GetAgentSkills(req, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("Failed to get agent skills, err: %v", err)
		c.JSON(code, utility.BuildErrorResponse(code, "error", err.Error(), "failed to get agent skills", nil))
		return
	}
	c.JSON(code, utility.BuildSuccessResponse(code, "Agent skills retrieved", skills, pagination))
}

func (base *Controller) GetAgentSkillByID(c *gin.Context) {

	var req models.CreateAgentSkillRequest
	agent_id := c.Param("agents_id")

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid skill_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent_id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.AgentId = agent_id

	skill_id := c.Param("skill_id")

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid skill_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid skill_id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.SkillId = skill_id

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)

	if _, err := uuid.Parse(req.OrgId); err != nil || req.OrgId == "" {
		base.Logger.Info("invalid organization id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	skills, err := agents.GetAgentSkillByID(req, base.Db.Postgresql)
	if err != nil {
		code := http.StatusBadRequest
		base.Logger.Error("Failed to fetch agent skill, err: %v", err)
		c.JSON(code, utility.BuildErrorResponse(code, "error", err.Error(), "failed to get agent skills", nil))
		return
	}
	code := http.StatusOK
	c.JSON(code, utility.BuildSuccessResponse(code, "Agent skill retrieved", skills))
}

func (base *Controller) UpdateAgentSkill(c *gin.Context) {
	skill_id := c.Param("skill_id")
	var (
		updateData models.UpdateAgentSkillRequest
		req        models.CreateAgentRequest
	)

	if _, err := uuid.Parse(skill_id); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid skill id format", "failed to decode skill id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	agent_id := c.Param("agents_id")

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid skill_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent_id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid request body", err, nil))
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)
	req.UserId = userClaims["user_id"].(string)
	req.AgentId = agent_id
	req.SkillId = skill_id

	if _, err := uuid.Parse(req.OrgId); err != nil || req.OrgId == "" {
		base.Logger.Info("invalid organization id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	updated, err := agents.UpdateAgentSkill(req, updateData, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to update agent skill, err: %v", err)
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to update agent skill", nil))
		return
	}
	c.JSON(http.StatusOK, utility.BuildSuccessResponse(http.StatusOK, "Agent skill updated", updated))
}

func (base *Controller) DeleteAgentSkill(c *gin.Context) {

	var req models.CreateAgentRequest

	skillID := c.Param("skill_id")

	if _, err := uuid.Parse(skillID); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid skill id format", "failed to decode skill id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	agentID := c.Param("agents_id")
	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Error("invalid skill_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent_id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)
	req.UserId = userClaims["user_id"].(string)
	req.AgentId = agentID
	req.SkillId = skillID

	if _, err := uuid.Parse(req.OrgId); err != nil || req.OrgId == "" {
		base.Logger.Info("invalid organization id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := agents.DeleteAgentSkill(req, base.Db.Postgresql); err != nil {
		base.Logger.Error("Failed to delete agent skill, err: %v", err)
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to delete agent skill", nil))
		return
	}
	c.JSON(http.StatusOK, utility.BuildSuccessResponse(http.StatusOK, "Agent skill deleted", nil))
}

func (base *Controller) GetGeneralAgentSkill(c *gin.Context) {

	skills, pagination, err, code := agents.GetGeneralAgentSkills(base.Db.Postgresql, c)
	if err != nil {
		c.JSON(code, utility.BuildErrorResponse(code, "error", err.Error(), "failed to get agent skills", nil))
		return
	}
	c.JSON(code, utility.BuildSuccessResponse(code, "Agent skills retrieved", skills, pagination))
}

func (base *Controller) GetGeneralAgentSkillByID(c *gin.Context) {
	agent_id := c.Param("skill_id")

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid skill_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid skill_id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	skills, err := agents.GetGeneralAgentSkillByID(agent_id, base.Db.Postgresql)
	if err != nil {
		code := http.StatusBadRequest
		c.JSON(code, utility.BuildErrorResponse(code, "error", err.Error(), "failed to get agent skills", nil))
		return
	}
	code := http.StatusOK
	c.JSON(code, utility.BuildSuccessResponse(code, "Agent skill retrieved", skills))
}

func (base *Controller) AddSkillsToAgent(c *gin.Context) {
	var req models.CreateAgentSkillsRequest
	agent_id := c.Param("agents_id")

	if _, err := uuid.Parse(agent_id); err != nil {
		base.Logger.Error("invalid skill_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent_id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid Request body")
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid request body", err, nil))
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)
	req.UserId = userClaims["user_id"].(string)

	if _, err := uuid.Parse(req.OrgId); err != nil || req.OrgId == "" {
		base.Logger.Info("invalid organization id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.AgentId = agent_id

	code, err := agents.AddSkillToAgent(req, base.Db.Postgresql, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to add agent skills", err)
		c.JSON(code, utility.BuildErrorResponse(code, "error", err.Error(), "failed to add skill to agent", nil))
		return
	}

	base.Logger.Info("Agent skill added successfully")
	rd := utility.BuildSuccessResponse(code, "Skill added to agent successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) SearchSkills(c *gin.Context) {
	skills, paginationResponse, err, code := agents.SearchSkillsService(c, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to search skills", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to search skills", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(*skills),
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "skills retrieved successfully.", skills, paginationData)
	c.JSON(http.StatusOK, rd)
}
