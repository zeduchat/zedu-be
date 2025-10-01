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

func (base *Controller) CreateAgentTasks(c *gin.Context) {
	var req models.CreateAgentTasksRequest

	agentID := c.Param("agent_id")

	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Info("invalid agent id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", err, nil)
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
	orgID := userClaims["org_id"].(string)

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

	req.AgentID = agentID
	req.OrganisationID = orgID
	code, tasks, err := agents.CreateAgentTasks(base.Db.Postgresql, base.Logger, req)
	if err != nil {
		base.Logger.Error("error creating tasks", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Tasks created successfully")
	rd := utility.BuildSuccessResponse(code, "Tasks created successfully", tasks)
	c.JSON(code, rd)
}

func (base *Controller) UpdateAgentTasks(c *gin.Context) {
	var req models.UpdateAgentTasksRequest

	agentID := c.Param("agent_id")
	taskID := c.Param("task_id")

	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Info("invalid agent id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(taskID); err != nil {
		base.Logger.Info("invalid task id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid task id format", err, nil)
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

	ids := models.IDS{
		AgentID: agentID,
		TaskID:  taskID,
	}

	code, err := agents.UpdateAgentTasks(base.Db.Postgresql, base.Logger, req, ids)
	if err != nil {
		base.Logger.Error("error updating tasks", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Tasks updated successfully")
	rd := utility.BuildSuccessResponse(code, "Tasks updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) GetAgentTasks(c *gin.Context) {
	agentID := c.Param("agent_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	orgID := userClaims["org_id"].(string)

	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Info("invalid agent workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent workflow id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, code, err := agents.GetAgentTasks(c, base.Db.Postgresql, base.Logger, agentID, orgID)
	if err != nil {
		base.Logger.Error("error fetching tasks", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Tasks fetched successfully")
	c.JSON(code, utility.BuildSuccessResponse(code, "Tasks fetched successfully", resp))
}

func (base *Controller) DeleteAgentTasks(c *gin.Context) {
	agentID := c.Param("agent_id")
	taskID := c.Param("task_id")

	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Info("invalid agent workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent workflow id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	if _, err := uuid.Parse(taskID); err != nil {
		base.Logger.Info("invalid task id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid task id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		AgentID: agentID,
		TaskID:  taskID,
	}

	code, err := agents.DeleteAgentTasks(c, base.Db.Postgresql, base.Logger, ids)
	if err != nil {
		base.Logger.Error("error deleting task", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Task deleted successfully")
	rd := utility.BuildSuccessResponse(code, "Task deleted successfully", nil)
	c.JSON(code, rd)

}

func (base *Controller) ProcessAgentTasks(c *gin.Context) {
	agentID := c.Param("agent_id")

	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Info("invalid agent workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent workflow id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "invalid user claims type", nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	userID, ok := userClaims["user_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id must be string", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	
	orgID, ok := userClaims["org_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "org_id must be string", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		OrganisationID: orgID,
		AgentID:        agentID,
		UserID:         userID,
	}

	code, resp, err := agents.ProcessAgentTasks(c, base.Db.Postgresql, base.Logger, base.ExtReq, ids)
	if err != nil {
		base.Logger.Error("error processing tasks", err)
		rd := utility.BuildErrorResponse(code, "error", "error processing agent tasks", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Tasks processed successfully: Recommendations sent to agent and Workflow successfully generated")
	rd := utility.BuildSuccessResponse(code, "Tasks processed successfully: Recommendations sent to agent and Workflow successfully generated", resp)
	c.JSON(code, rd)
}
