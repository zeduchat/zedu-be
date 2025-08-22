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

	wGroup := r.Group(fmt.Sprintf("%v/channel-workflows", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		wGroup.POST("", wfCtrl.AddWorkflowToChannel)
		wGroup.DELETE("/:workflow_id/channels/:channel_id", wfCtrl.RemoveWorkflowFromChannel)
		wGroup.GET("/organisations/:org_id/channels/:channel_id", wfCtrl.GetChannelWorkflows)
	}

	wGMPGroup := r.Group(fmt.Sprintf("%v/workflows", ApiVersion))
	{

		wGMPGroup.GET("/:workflow_id", wfCtrl.GetGeneralMarketPlaceWorkflowByID)
		wGMPGroup.GET("/", wfCtrl.GetGeneralMarketWorkflows)
	}

	//workflowtasks
	workflowsCtrl := workflow.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	workflowURL := r.Group(fmt.Sprintf("%v/workflow", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		workflowURL.PUT("/:workflow_id/tasks", workflowsCtrl.UpdateWorkflowTasks)
		workflowURL.GET("/:workflow_id/tasks", workflowsCtrl.GetWorkflowTasks)
		workflowURL.GET("/:workflow_id/skills", workflowsCtrl.GetWorkflowSkills)
	}


	return r
}
