package workflow

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/workflow"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) UpdateWorkflowTasks(c *gin.Context) {
	var req models.UpdateWorkflowTasksRequest

	workflowID := c.Param("workflow_id")

	if _, err := uuid.Parse(workflowID); err != nil {
		base.Logger.Info("invalid workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid workflow id format", err, nil)
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

	req.WorkflowID = workflowID
	code,resp, err := workflow.UpdateWorkflowTasks(base.Db.Postgresql, req)
	if err != nil {
		base.Logger.Error("error creating tasks", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Tasks created successfully")
	rd := utility.BuildSuccessResponse(code, "Tasks created successfully", resp)
	c.JSON(code, rd)
}
