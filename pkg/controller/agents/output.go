package agents

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) FetchOutputAgents(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	agents, err := agents.GetActiveOutputIntegrations(base.Db.Postgresql, org_id)
	if err != nil {
		base.Logger.Error("Failed to get out putagents")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to get output agents", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	base.Logger.Info("output agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "output agents retrieved successfully.", agents)
	c.JSON(http.StatusOK, rd)
}

// Fetch System Integration without org details (Unauthorized)
func (base *Controller) GetSystemAgentApps(c *gin.Context) {

	agents, paginationResponse, err, code := agents.GetSystemAgentApps(c, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch agents", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch agents", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents retrieved successfully.", agents, paginationData)
	c.JSON(http.StatusOK, rd)
}

// Get Single Integration App
func (base *Controller) GetSystemAgentApp(c *gin.Context) {

	int_id := c.Param("agent_id")

	if _, err := uuid.Parse(int_id); err != nil {
		parts := strings.Split(int_id, "-")
		if len(parts) != 2 {
			base.Logger.Error("invalid organisation id format", err)
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		int_id = parts[1]
	}

	agent, err, code := agents.GetSystemAgentApp(c, base.Db.Postgresql, int_id, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch agents", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch agents", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents retrieved successfully.", agent)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) TriggerTick(c *gin.Context) {
	var req models.TriggerTickRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	response, code, err := agents.TriggerTick(base.Db, base.Logger, req)
	if err != nil {
		base.Logger.Error("Failed to trigger tick", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to trigger tick", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("tick called successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "tick called successfully", response)
	c.JSON(http.StatusOK, rd)
}

// Search Agents (featured, category, or keyword search)
func (base *Controller) SearchAgents(c *gin.Context) {
	agents, paginationResponse, err, code := agents.SearchAgentsService(c, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to search agents", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to search agents", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("agents retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "agents retrieved successfully.", agents, paginationData)
	c.JSON(http.StatusOK, rd)
}
