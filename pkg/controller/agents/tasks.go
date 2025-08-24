package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/agents"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) UpdateAgentTasks(c *gin.Context) {
	var req models.UpdateAgentTasksRequest

	agentID := c.Param("agent_id")

	if _, err := uuid.Parse(agentID); err != nil {
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

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	req.AgentID = agentID
	code, tasks, err := agents.UpdateAgentTasks(c, base.Db.Postgresql, base.Logger, base.ExtReq, req)
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

func (base *Controller) GetAgentTasks(c *gin.Context) {
	agentID := c.Param("agent_id")

	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Info("invalid agent workflow id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent workflow id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, code, err := agents.GetAgentTasks(c, base.Db.Postgresql, base.Logger, agentID)
	if err != nil {
		base.Logger.Error("error fetching tasks", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Tasks fetched successfully")
	c.JSON(code, utility.BuildSuccessResponse(code, "Tasks fetched successfully", resp))
}

