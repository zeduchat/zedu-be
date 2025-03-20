package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/integrations"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Agents(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	integrationsCtrl := integrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	organisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		// Organisation integrations routes
		organisationUrl.GET("/:org_id/agents", integrationsCtrl.GetAllAgentApp)
		organisationUrl.PATCH("/:org_id/agents/:agent_id", integrationsCtrl.UpdateAgentApp)
		organisationUrl.DELETE("/:org_id/agents/:agent_id", integrationsCtrl.DeleteAgentApp)
		organisationUrl.PATCH("/:org_id/agents/change_status", integrationsCtrl.ChangeAgentStatus)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/updatejson", integrationsCtrl.UpdateJSONSchema)
		organisationUrl.GET("/:org_id/agents/output", integrationsCtrl.FetchOutputAgents)

		// Organization Custom Agents
		organisationUrl.POST("/:org_id/agents/custom", integrationsCtrl.CreateCustomAgent)
		organisationUrl.DELETE("/:org_id/agents/custom/:agent_id", integrationsCtrl.DeleteCustomAgentApp)
		organisationUrl.GET("/:org_id/agents/custom", integrationsCtrl.GetCustomAgentApp)
		organisationUrl.PUT("/:org_id/agents/custom/:agent_id", integrationsCtrl.UpdateCustomAgent)
		organisationUrl.GET("/:org_id/agents/custom/:agent_id/settings", integrationsCtrl.GetCustomAgentSettings)
		organisationUrl.GET("/:org_id/agents/custom/:agent_id/status", integrationsCtrl.GetCustomAgentStatus)
		organisationUrl.PUT("/:org_id/agents/custom/:agent_id/settings", integrationsCtrl.UpdateCustomAgentSettings)

		// Channel agents routes
		organisationUrl.GET("/:org_id/channels/:channel_id/agents", integrationsCtrl.GetOrganisationChannelAgents)
		organisationUrl.PATCH("/:org_id/channels/:channel_id/agents/change-sendback-status", integrationsCtrl.ChangeOrgChannelIntSendBackStatus)
		organisationUrl.POST("/:org_id/agents/:agent_id/channels/:channel_id", integrationsCtrl.ActivateDeactivateChannelAgent)
		organisationUrl.GET("/:org_id/agents/:agent_id/channels", integrationsCtrl.AgentChannels)
		organisationUrl.GET("/:org_id/agents/:agent_id/status", integrationsCtrl.CheckAgentIsActive)

		// Organisation agent settings routes
		organisationUrl.POST("/:org_id/agents/:agent_id/settings", integrationsCtrl.AddAgentSetting)
		organisationUrl.GET("/:org_id/agents/:agent_id/settings", integrationsCtrl.GetAgentSettings)
		organisationUrl.GET("/:org_id/agents/:agent_id/integration-api-key", integrationsCtrl.GetAgentSettings)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/settings/:setting_id", integrationsCtrl.UpdateAgentSetting)

		// Organisation channel agent settings routes
		organisationUrl.POST("/:org_id/agents/:agent_id/channels/:channel_id/settings", integrationsCtrl.AddChannelAgentSetting)
		organisationUrl.GET("/:org_id/agents/:agent_id/channels/:channel_id/settings", integrationsCtrl.GetChannelAgentSetting)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/channels/:channel_id/settings/:setting_id", integrationsCtrl.UpdateChannelAgentSetting)
		organisationUrl.DELETE("/:org_id/agents/:agent_id/channels/:channel_id/settings/:setting_id", integrationsCtrl.DeleteChannelAgentSetting)

		// Agent slash commands routes
		organisationUrl.POST("/:org_id/agents/:agent_id/slash-commands", integrationsCtrl.AddAgentSlashCommand)
		organisationUrl.GET("/:org_id/agents/:agent_id/slash-commands", integrationsCtrl.GetAgentSlashCommands)
		organisationUrl.GET("/:org_id/slash-commands", integrationsCtrl.GetAllOrgSlashCommands)
		organisationUrl.PATCH("/:org_id/agents/:agent_id/slash-commands/:command_id", integrationsCtrl.UpdateAgentSlashCommand)
		organisationUrl.DELETE("/:org_id/agents/:agent_id/slash-commands/:command_id", integrationsCtrl.DeleteAgentSlashCommand)
	}

	integration := integrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	integrationUrl := r.Group(fmt.Sprintf("%v/agents", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		integrationUrl.GET("/:agent_id/settings", integration.GetAgentSettingsAllOrgs)
		integrationUrl.POST("/trigger-tick", integration.TriggerTick)
	}

	// Unauthenticated endpoint to fetch integrations
	intPage := r.Group(fmt.Sprintf("%v/agents", ApiVersion))
	{
		intPage.GET("", integration.GetSystemAgentApps)
		intPage.GET("/:agent_id", integration.GetSystemAgentApp)
	}

	external_int := r.Group(fmt.Sprintf("%v/agents/settings", ApiVersion))
	{
		external_int.GET("", integration.GetCustomAgentSettingsExteranl)
		external_int.PUT("", integration.UpdateCustomAgentSettingsExternal)
	}

	return r
}
