package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	dm "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/controller/dm_filter"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Organisation(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	organisationCtrl := organisation.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	channelCtrl := channel.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	integrationsCtrl := agents.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	dmCtrl := dm.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	dmFilter := dm_filter.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	organisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		// Organisation routes
		organisationUrl.POST("", organisationCtrl.CreateOrganisation)
		organisationUrl.GET("/:org_id", organisationCtrl.GetOrganisation)
		organisationUrl.DELETE("/:org_id", organisationCtrl.DeleteOrganisation)
		organisationUrl.PUT("/:org_id", organisationCtrl.UpdateOrganisation)

		// Organisation roles routes
		organisationUrl.POST("/:org_id/roles", organisationCtrl.CreateOrgRole)
		organisationUrl.GET("/:org_id/roles", organisationCtrl.GetOrgRoles)
		organisationUrl.GET("/:org_id/roles/:role_id", organisationCtrl.GetAOrgRole)
		organisationUrl.DELETE("/:org_id/roles/:role_id", organisationCtrl.DeleteOrgRole)
		organisationUrl.PUT("/:org_id/roles/:role_id", organisationCtrl.UpdateOrgRole)
		organisationUrl.PUT("/:org_id/roles/:role_id/permissions", organisationCtrl.UpdateOrgPermissions)

		// Organisation channels routes
		organisationUrl.GET("/:org_id/channels", organisationCtrl.GetAllChannelssInOrganisation)
		organisationUrl.GET("/:org_id/user-channels", channelCtrl.GetUserChannels)
		organisationUrl.GET("/:org_id/user-not-channels", channelCtrl.GetUserNotInChannels)
		organisationUrl.GET("/:org_id/channels/archived", channelCtrl.GetArchivedChannels)
		organisationUrl.PUT("/:org_id/archive-channel", channelCtrl.ArchiveChannel)

		// User management routes
		organisationUrl.GET("/:org_id/users", organisationCtrl.GetUsersInOrganisation)
		organisationUrl.GET("/:org_id/metrics", organisationCtrl.GetOrganisationCountMetrics)
		organisationUrl.DELETE("/:org_id/users/:user_id", organisationCtrl.RemoveMemberFromOrganisation)
		organisationUrl.PUT("/:org_id/users/:user_id", organisationCtrl.UpdateMember)
		organisationUrl.GET("/:org_id/invites", organisationCtrl.GetOrganisationInvites)
		organisationUrl.POST("/:org_id/users", organisationCtrl.AddMemberToOrganisation)

		// Organisation integrations routes
		organisationUrl.GET("/:org_id/integrations", integrationsCtrl.GetAllAgentApp)
		organisationUrl.PATCH("/:org_id/integrations/:agent_id", integrationsCtrl.UpdateAgentApp)
		organisationUrl.DELETE("/:org_id/integrations/:agent_id", integrationsCtrl.DeleteAgentApp)
		organisationUrl.PATCH("/:org_id/integrations/change_status", integrationsCtrl.ChangeAgentStatus)
		organisationUrl.PATCH("/:org_id/integrations/:agent_id/updatejson", integrationsCtrl.UpdateJSONSchema)
		organisationUrl.GET("/:org_id/integrations/output", integrationsCtrl.FetchOutputAgents)

		// Organization Custom Agents
		organisationUrl.POST("/:org_id/integrations/custom", integrationsCtrl.CreateCustomAgent)
		organisationUrl.DELETE("/:org_id/integrations/custom/:agent_id", integrationsCtrl.DeleteCustomAgentApp)
		organisationUrl.GET("/:org_id/integrations/custom", integrationsCtrl.GetCustomAgentApp)
		organisationUrl.PUT("/:org_id/integrations/custom/:agent_id", integrationsCtrl.UpdateCustomAgent)
		organisationUrl.GET("/:org_id/integrations/custom/:agent_id/settings", integrationsCtrl.GetCustomAgentSettings)
		organisationUrl.GET("/:org_id/integrations/custom/:agent_id/status", integrationsCtrl.GetCustomAgentStatus)
		organisationUrl.PUT("/:org_id/integrations/custom/:agent_id/settings", integrationsCtrl.UpdateCustomAgentSettings)

		// Channel integrations routes
		organisationUrl.GET("/:org_id/channels/:channel_id/integrations", integrationsCtrl.GetOrganisationChannelAgents)
		organisationUrl.PATCH("/:org_id/channels/:channel_id/integrations/change-sendback-status", integrationsCtrl.ChangeOrgChannelIntSendBackStatus)
		organisationUrl.POST("/:org_id/integrations/:agent_id/channels/:channel_id", integrationsCtrl.ActivateDeactivateChannelAgent)
		organisationUrl.GET("/:org_id/integrations/:agent_id/channels", integrationsCtrl.AgentChannels)
		organisationUrl.GET("/:org_id/integrations/:agent_id/status", integrationsCtrl.CheckAgentIsActive)

		// Organisation integration settings routes
		organisationUrl.POST("/:org_id/integrations/:agent_id/settings", integrationsCtrl.AddAgentSetting)
		organisationUrl.GET("/:org_id/integrations/:agent_id/settings", integrationsCtrl.GetAgentSettings)
		organisationUrl.GET("/:org_id/integrations/:agent_id/integration-api-key", integrationsCtrl.GetAgentSettings)
		organisationUrl.PATCH("/:org_id/integrations/:agent_id/settings/:setting_id", integrationsCtrl.UpdateAgentSetting)

		// Organisation channel integration settings routes
		organisationUrl.POST("/:org_id/integrations/:agent_id/channels/:channel_id/settings", integrationsCtrl.AddChannelAgentSetting)
		organisationUrl.GET("/:org_id/integrations/:agent_id/channels/:channel_id/settings", integrationsCtrl.GetChannelAgentSetting)
		organisationUrl.PATCH("/:org_id/integrations/:agent_id/channels/:channel_id/settings/:setting_id", integrationsCtrl.UpdateChannelAgentSetting)
		organisationUrl.DELETE("/:org_id/integrations/:agent_id/channels/:channel_id/settings/:setting_id", integrationsCtrl.DeleteChannelAgentSetting)

		// Agent slash commands routes
		organisationUrl.POST("/:org_id/integrations/:agent_id/slash-commands", integrationsCtrl.AddAgentSlashCommand)
		organisationUrl.GET("/:org_id/integrations/:agent_id/slash-commands", integrationsCtrl.GetAgentSlashCommands)
		organisationUrl.GET("/:org_id/slash-commands", integrationsCtrl.GetAllOrgSlashCommands)
		organisationUrl.PATCH("/:org_id/integrations/:agent_id/slash-commands/:command_id", integrationsCtrl.UpdateAgentSlashCommand)
		organisationUrl.DELETE("/:org_id/integrations/:agent_id/slash-commands/:command_id", integrationsCtrl.DeleteAgentSlashCommand)

		// DM endpoints
		organisationUrl.POST("/:org_id/dms", dmCtrl.CreateDmChannel)
		organisationUrl.DELETE("/:org_id/dms/:channel_id", dmCtrl.DeleteDmChannel)
		organisationUrl.GET("/:org_id/dms", dmCtrl.GetDmChannels)
		organisationUrl.GET("/:org_id/dms/participants/:channel_id", dmCtrl.GetDmParticipants)
		organisationUrl.GET("/:org_id/recent-dm", dmFilter.DmFilter)

		// Group DM endpoints
		organisationUrl.POST("/:org_id/group-dms", dmCtrl.CreateGroupDMChannel)
		organisationUrl.DELETE("/:org_id/group-dms/:channel_id", dmCtrl.DeleteGroupDMChannel)
		organisationUrl.GET("/:org_id/group-dms", dmCtrl.GetGroupDMChannels)
		organisationUrl.GET("/:org_id/group-dms/:user_id", dmCtrl.GetUserGroupDMs)

		//bots
		organisationUrl.GET("/:org_id/fetch-bots", integrationsCtrl.FetchOrganisationBots)

	}

	// Test routes
	testOrganisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion))
	{
		testOrganisationUrl.GET("/:org_id/load-metrics", organisationCtrl.GetLoadingMetrics)
	}

	return r
}
