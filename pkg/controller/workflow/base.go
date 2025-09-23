package workflow

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/services/workflow"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateWorkflow(c *gin.Context) {
	var req models.WorkFlowRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.UserId = userClaims["user_id"].(string)
	req.OrgId = c.Param("org_id")

	if _, err := uuid.Parse(req.OrgId); err != nil {
		base.Logger.Info("invalid organization id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := workflow.CreateWorkflowService(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error creating workflow", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow created successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow created successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) GetAllWorkflowsByOrg(c *gin.Context) {
	var req models.WorkFlowRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	req.UserId = userClaims["user_id"].(string)
	req.OrgId = c.Param("org_id")

	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organization id format", errors.New("failed to parse organization id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, code, err := workflow.ListWorkflowsService(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflows retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "Workflows retrieved successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) GetWorkflowByID(c *gin.Context) {
	var req models.WorkFlowRequest

	req.Id = c.Param("workflow_id")
	req.OrgId = c.Param("org_id")

	if _, err := uuid.Parse(req.Id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid workflow ID format", errors.New("failed to parse workflow ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org ID format", errors.New("failed to parse org ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.UserId = userClaims["user_id"].(string)

	resp, code, err := workflow.GetWorkflowByIDService(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow retrieved successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) DeleteWorkflow(c *gin.Context) {
	var req models.WorkFlowRequest

	req.Id = c.Param("workflow_id")
	req.OrgId = c.Param("org_id")

	if _, err := uuid.Parse(req.Id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid workflow ID format", errors.New("failed to parse workflow ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org ID format", errors.New("failed to parse org ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	req.UserId = userClaims["user_id"].(string)

	code, err := workflow.DeleteWorkflowService(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow deleted successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow deleted successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) UpdateWorkflow(c *gin.Context) {
	var req models.WorkFlowRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	req.UserId = userClaims["user_id"].(string)
	req.OrgId = c.Param("org_id")
	req.Id = c.Param("workflow_id")

	if _, err := uuid.Parse(req.Id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid workflow ID format", errors.New("invalid workflow ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(req.OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation ID format", errors.New("invalid org ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := workflow.UpdateWorkflowService(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow updated successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) AddWorkflowToChannel(c *gin.Context) {
	var req models.ChannelWorkflowRequest

	_, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := workflow.UpdateWorkflowStatus(base.Db.Postgresql, req)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Channel workflow updated successfully")
	rd := utility.BuildSuccessResponse(code, "Channel workflow updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) RemoveWorkflowFromChannel(c *gin.Context) {
	var req models.ChannelWorkflowRequest

	req.WorkflowID = c.Param("workflow_id")
	req.ChannelID = c.Param("channel_id")

	if _, err := uuid.Parse(req.ChannelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel ID format", errors.New("failed to parse channel ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(req.WorkflowID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid workflow ID format", errors.New("failed to parse oworkflow ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	_, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := workflow.RemoveWorkflowFromChannel(base.Db.Postgresql, req)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow removed successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow removed successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) GetChannelWorkflows(c *gin.Context) {

	ChannelID := c.Param("channel_id")
	OrgId := c.Param("org_id")
	if _, err := uuid.Parse(OrgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation ID format", errors.New("invalid org ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(ChannelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel ID format", errors.New("failed to parse channel ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	_, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, code, err := workflow.GetChannelWorkflows(base.Db.Postgresql, &ChannelID, &OrgId)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow fetched successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow fetched successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) GetGeneralMarketPlaceWorkflowByID(c *gin.Context) {
	var req models.WorkFlowRequest

	req.Id = c.Param("workflow_id")

	if _, err := uuid.Parse(req.Id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid workflow ID format", errors.New("failed to parse workflow ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	req.UserId = userClaims["user_id"].(string)

	resp, code, err := workflow.GetGeneralMarketPlaceWorkflowId(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow retrieved successfully", resp)
	c.JSON(code, rd)
}

func (base *Controller) GetGeneralMarketWorkflows(c *gin.Context) {

	resp, pag, code, err := workflow.ListGeneralMarketPlaceWorkflows(base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Workflow fetched successfully")
	rd := utility.BuildSuccessResponse(code, "Workflow fetched successfully", resp, pag)
	c.JSON(code, rd)
}

func (base *Controller) SearchWorkflows(c *gin.Context) {
	workflows, paginationResponse, err, code := agents.SearchWorkflowsService(c, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to search workflows", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to search workflows", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "workflows retrieved successfully.", workflows, paginationData)
	c.JSON(http.StatusOK, rd)
}
