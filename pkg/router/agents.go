package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func AgentSkill(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	agentsCtrl := agents.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	organisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		organisationUrl.POST("/:org_id/agents/:agents_id/integration", agentsCtrl.CreateAgentSkill)
		organisationUrl.GET("/:org_id/agents/:agents_id/integration", agentsCtrl.GetAgentSkill)
		organisationUrl.PUT("/:org_id/agents/:agents_id/integration/:integration_id", agentsCtrl.UpdateAgentSkill)
		organisationUrl.DELETE("/:org_id/agents/:agents_id/integration/:integration_id", agentsCtrl.DeleteAgentSkill)
	}

	return r
}

// func AgentTasks(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
// 	extReq := request.ExternalRequest{Logger: logger, Test: false}
// 	agentsCtrl := agents.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
// 	organisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))

// 	{
// 		organisationUrl.POST("/:org_id/agents/:agents_id/task", agentsCtrl.CreateAgentTask)
// 		organisationUrl.GET("/:org_id/agents/:agents_id/task", agentsCtrl.GetAgentTasks)
// 		organisationUrl.PUT("/:org_id/agents/:agents_id/task/:task_id", agentsCtrl.UpdateAgentTask)
// 		organisationUrl.DELETE("/:org_id/agents/:agents_id/task/:task_id", agentsCtrl.DeleteAgentTask)
// 	}

// 	return r
// }


func Agents(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	agentsCtrl := agents.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	organisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		// Organisation agents routes
		organisationUrl.GET("/:org_id/agents/system", agentsCtrl.GetAllAgentApp)
		organisationUrl.PATCH("/:org_id/agents/system/:agent_id", agentsCtrl.UpdateAgentApp)
		organisationUrl.DELETE("/:org_id/agents/system/:agent_id", agentsCtrl.DeleteAgentApp)
		organisationUrl.PATCH("/:org_id/agents/change_status", agentsCtrl.ChangeAgentStatus)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/updatejson", agentsCtrl.UpdateJSONSchema)
		organisationUrl.GET("/:org_id/agents/output", agentsCtrl.FetchOutputAgents)

		// Organization Custom Agents
		organisationUrl.POST("/:org_id/agents", agentsCtrl.CreateCustomAgent)
		organisationUrl.GET("/:org_id/agents", agentsCtrl.FetchOrganisationBots)
		organisationUrl.PUT("/:org_id/agents/:agent_id", agentsCtrl.UpdateCustomAgent)
		organisationUrl.DELETE("/:org_id/agents/:agent_id", agentsCtrl.DeleteCustomAgentApp)

		organisationUrl.GET("/:org_id/agents/:agent_id/settings", agentsCtrl.GetCustomAgentSettings)
		organisationUrl.GET("/:org_id/agents/:agent_id/status", agentsCtrl.GetCustomAgentStatus)
		organisationUrl.PUT("/:org_id/agents/:agent_id/settings", agentsCtrl.UpdateCustomAgentSettings)

		// Channel agents routes
		organisationUrl.GET("/:org_id/channels/:channel_id/agents", agentsCtrl.GetOrganisationChannelAgents)
		organisationUrl.PATCH("/:org_id/channels/:channel_id/agents/change-sendback-status", agentsCtrl.ChangeOrgChannelIntSendBackStatus)
		organisationUrl.POST("/:org_id/agents/:agent_id/channels/:channel_id", agentsCtrl.ActivateDeactivateChannelAgent)
		organisationUrl.GET("/:org_id/agents/:agent_id/channels", agentsCtrl.AgentChannels)
		organisationUrl.GET("/:org_id/channels/agents/:agent_id/status", agentsCtrl.CheckAgentIsActive)

		// Organisation agent settings routes
		organisationUrl.POST("/:org_id/agents/:agent_id/settings", agentsCtrl.AddAgentSetting)
		organisationUrl.GET("/:org_id/settings/agents/:agent_id", agentsCtrl.GetAgentSettings)
		organisationUrl.GET("/:org_id/agents/:agent_id/agent-api-key", agentsCtrl.GetAgentSettings)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/settings/:setting_id", agentsCtrl.UpdateAgentSetting)

		// Organisation channel agent settings routes
		organisationUrl.POST("/:org_id/agents/:agent_id/channels/:channel_id/settings", agentsCtrl.AddChannelAgentSetting)
		organisationUrl.GET("/:org_id/agents/:agent_id/channels/:channel_id/settings", agentsCtrl.GetChannelAgentSetting)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/channels/:channel_id/settings/:setting_id", agentsCtrl.UpdateChannelAgentSetting)
		organisationUrl.DELETE("/:org_id/agents/:agent_id/channels/:channel_id/settings/:setting_id", agentsCtrl.DeleteChannelAgentSetting)

		// Agent slash commands routes
		organisationUrl.POST("/:org_id/agents/:agent_id/slash-commands", agentsCtrl.AddAgentSlashCommand)
		organisationUrl.GET("/:org_id/agents/:agent_id/slash-commands", agentsCtrl.GetAgentSlashCommands)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/slash-commands/:command_id", agentsCtrl.UpdateAgentSlashCommand)
		organisationUrl.DELETE("/:org_id/agents/:agent_id/slash-commands/:command_id", agentsCtrl.DeleteAgentSlashCommand)
	}

	agent := agents.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	agentUrl := r.Group(fmt.Sprintf("%v/agents", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		agentUrl.GET("/:agent_id/settings", agent.GetAgentSettingsAllOrgs)
		agentUrl.GET("/me", agentsCtrl.GetAgentsByOwner)
		agentUrl.GET("/:agent_id/activated-organizations", agent.GetActivatedOrganizations)
		agentUrl.POST("/trigger-tick", agent.TriggerTick)
	}

	// fetch all agents ---> will be used to get all agents on the superAdmin dashboard
	adminAgent := agents.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	adminAgentUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.AdminAuthorize(db.Postgresql))
	{
		adminAgentUrl.POST("/agents/system", agentsCtrl.CreateSystemAgent)
		adminAgentUrl.GET("/agents/all", adminAgent.GetAllCustomAgent)
		adminAgentUrl.GET("/agents/bills", adminAgent.GetAgentBills)
		adminAgentUrl.PUT("/agents/:agent_id", agentsCtrl.AdminUpdateAgent)
		adminAgentUrl.GET("/agents/bills/:org_id", adminAgent.GetOrgAgentBills)
		adminAgentUrl.GET("/agents/:source/:agent_id", adminAgent.GetCustomAgentByID)
		adminAgentUrl.GET("/agents/metrics", adminAgent.GetCustomAgentMetrics)
		adminAgentUrl.DELETE("/agents/:agent_id", agentsCtrl.AdminDeleteCustomAgentApp)
		adminAgentUrl.DELETE("/agents/system/:agent_id", agentsCtrl.AdminDeleteSystemAgentApp)
	}

	// Unauthenticated endpoint to fetch agents
	intPage := r.Group(fmt.Sprintf("%v/agents", ApiVersion))
	{
		intPage.GET("", agent.GetSystemAgentApps)
		intPage.GET("/callback", agent.AgentCallback)
		intPage.GET("/:agent_id", agent.GetSystemAgentApp)
	}

	external_int := r.Group(fmt.Sprintf("%v/agents/settings", ApiVersion))
	{
		external_int.GET("", agent.GetCustomAgentSettingsExteranl)
		external_int.PUT("", agent.UpdateCustomAgentSettingsExternal)
	}

	return r
}

