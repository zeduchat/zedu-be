package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/workflow"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func WorkflowRoutes(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	wfCtrl := workflow.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	orgGroup := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		orgGroup.POST("/:org_id/workflows", wfCtrl.CreateWorkflow)
		orgGroup.GET("/:org_id/workflows", wfCtrl.GetAllWorkflowsByOrg)
		orgGroup.GET("/:org_id/workflows/:workflow_id", wfCtrl.GetWorkflowByID)
		orgGroup.PUT("/:org_id/workflows/:workflow_id", wfCtrl.UpdateWorkflow)
		orgGroup.DELETE("/:org_id/workflows/:workflow_id", wfCtrl.DeleteWorkflow)
	}

	agentGroup := r.Group(fmt.Sprintf("%v/agents", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		agentGroup.POST("/:agent_id/workflows", wfCtrl.CreateAgentWorkflow)
		agentGroup.GET("/:agent_id/workflows", wfCtrl.ListAgentWorkflows)
		agentGroup.GET("/:agent_id/workflows/:workflow_id", wfCtrl.GetAgentWorkflowByID)
		agentGroup.PUT("/:agent_id/workflows/:workflow_id", wfCtrl.UpdateAgentWorkflow)
		agentGroup.DELETE("/:agent_id/workflows/:workflow_id", wfCtrl.DeleteAgentWorkflow)
	}

	wGroup := r.Group(fmt.Sprintf("%v/channel-workflows", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		wGroup.POST("", wfCtrl.AddWorkflowToChannel)
		wGroup.DELETE("/:workflow_id/channels/:channel_id", wfCtrl.RemoveWorkflowFromChannel)
		wGroup.GET("/organisations/:org_id/channels/:channel_id", wfCtrl.GetChannelWorkflows)
	}

	wGMPGroup := r.Group(fmt.Sprintf("%v/workflows", ApiVersion))
	{
		wGMPGroup.GET("/:workflow_id", wfCtrl.GetGeneralMarketPlaceWorkflowByID)
		wGMPGroup.GET("/agents/:agent_id", wfCtrl.ListGeneralAgentWorkflows)
		wGMPGroup.GET("/", wfCtrl.GetGeneralMarketWorkflows)
		wGMPGroup.GET("/search", wfCtrl.SearchWorkflows)
	}

	return r
}
