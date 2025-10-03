package workflow

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/workflow"
	"github.com/hngprojects/telex_be/utility"
)

// Create Agent Workflow
func (base *Controller) CreateAgentWorkflow(c *gin.Context) {
	var req models.AgentWorkFlowRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
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

	req.AgentId = c.Param("agent_id")

	if _, err := uuid.Parse(req.AgentId); err != nil {
		base.Logger.Info("invalid agent id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := workflow.CreateAgentWorkflowService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error creating workflow", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Agent Workflow created successfully")
	rd := utility.BuildSuccessResponse(code, "Agent Workflow created successfully", resp)
	c.JSON(code, rd)
}

// Get Agent Workflow by ID
func (base *Controller) GetAgentWorkflowByID(c *gin.Context) {
	var req models.AgentWorkFlowRequest
	req.AgentId = c.Param("agent_id")
	req.WorkflowId = c.Param("workflow_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.AgentId); err != nil {
		base.Logger.Info("invalid agent id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.WorkflowId); err != nil {
		base.Logger.Info("invalid workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid workflow id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)

	resp, code, err := workflow.GetAgentWorkflowByIDService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error fetching workflow", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Agent Workflow fetched successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) ListAgentWorkflows(c *gin.Context) {
	var req models.AgentWorkFlowRequest
	req.AgentId = c.Param("agent_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.AgentId); err != nil {
		base.Logger.Info("invalid agent id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)

	resp, paginationResponse, code, err := workflow.ListAgentWorkflowsService(req, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("error listing workflows", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	rd := utility.BuildSuccessResponse(code, "Agent Workflows listed successfully", resp, paginationData)
	c.JSON(code, rd)
}

func (base *Controller) ListGeneralAgentWorkflows(c *gin.Context) {
	var req models.AgentWorkFlowRequest
	req.AgentId = c.Param("agent_id")

	if _, err := uuid.Parse(req.AgentId); err != nil {
		parts := strings.Split(req.AgentId, "-")
		if len(parts) < 2 || len(parts[len(parts)-1]) != 12 {
			base.Logger.Error("invalid agent id format", err)
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent id format", "failed to decode agent id", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		req.AgentId = parts[len(parts)-1]
	}

	req.IsPublic = true
	resp, paginationResponse, code, err := workflow.ListAgentWorkflowsService(req, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("error listing workflows", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	rd := utility.BuildSuccessResponse(code, "Agent Workflows listed successfully", resp, paginationData)
	c.JSON(code, rd)
}

func (base *Controller) DeleteAgentWorkflow(c *gin.Context) {
	var req models.AgentWorkFlowRequest
	req.AgentId = c.Param("agent_id")
	req.WorkflowId = c.Param("workflow_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.AgentId); err != nil {
		base.Logger.Info("invalid agent id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.WorkflowId); err != nil {
		base.Logger.Info("invalid workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid workflow id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)

	err, code := workflow.DeleteAgentWorkflowService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error deleting workflow", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Agent Workflow deleted successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) UpdateAgentWorkflow(c *gin.Context) {
	var req models.AgentWorkFloUpdateRequest
	req.AgentId = c.Param("agent_id")
	req.WorkflowId = c.Param("workflow_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.AgentId); err != nil {
		base.Logger.Info("invalid agent id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.WorkflowId); err != nil {
		base.Logger.Info("invalid workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid workflow id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	err, code := workflow.UpdateAgentWorkflowService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error updating workflow", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Agent Workflow updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) UpdateWorkflowNode(c *gin.Context) {
	var req models.AgentWorkFloNodeUpdateRequest

	node_id := c.Param("node_id")
	if _, err := uuid.Parse(node_id); err != nil {
		base.Logger.Error("invalid node_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid node_id format", "failed to decode workflow id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	
	workflow_id := c.Param("workflow_id")
	if _, err := uuid.Parse(workflow_id); err != nil {
		base.Logger.Error("invalid workflow_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid workflow_id format", "failed to decode workflow id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	
	// bind the json body to updateData struct and validate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid request body", err, nil))
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}
	
	if _, err := uuid.Parse(req.AgentId); err != nil {
		base.Logger.Error("invalid agent_id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid agent_id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.OrgId = userClaims["org_id"].(string)
	req.NodeID = node_id
	req.WorkflowId = workflow_id

	if _, err := uuid.Parse(req.OrgId); err != nil || req.OrgId == "" {
		base.Logger.Info("invalid organization id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	updated, err := workflow.UpdateWorkflowNodeService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to update workflow node, err: %v", err)
		c.JSON(http.StatusInternalServerError, utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to update agent skill", nil))
		return
	}

	c.JSON(http.StatusOK, utility.BuildSuccessResponse(http.StatusOK, "Workflow Node updated", updated))
}
