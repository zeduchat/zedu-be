package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	dm "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/controller/integrations"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Organisation(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	organisationCtrl := organisation.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	channelCtrl := channel.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	integrationsCtrl := integrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	dmCtrl := dm.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
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
		// organisationUrl.GET("/:org_id/integrations", integrationsCtrl.GetAllIntegrationApp)
		// organisationUrl.PATCH("/:org_id/integrations/:integration_id", integrationsCtrl.UpdateIntegrationApp)
		// organisationUrl.DELETE("/:org_id/integrations/:integration_id", integrationsCtrl.DeleteIntegrationApp)
		// organisationUrl.PATCH("/:org_id/integrations/change_status", integrationsCtrl.ChangeIntegrationStatus)
		// organisationUrl.PATCH("/:org_id/integrations/:integration_id/updatejson", integrationsCtrl.UpdateJSONSchema)
		// organisationUrl.GET("/:org_id/integrations/output", integrationsCtrl.FetchOutputIntegrations)

		// // Organization Custom Integrations
		// organisationUrl.POST("/:org_id/integrations/custom", integrationsCtrl.CreateCustomIntegration)
		// organisationUrl.DELETE("/:org_id/integrations/custom/:integration_id", integrationsCtrl.DeleteCustomIntegrationApp)
		// organisationUrl.GET("/:org_id/integrations/custom", integrationsCtrl.GetCustomIntegrationApp)
		// organisationUrl.PUT("/:org_id/integrations/custom/:integration_id", integrationsCtrl.UpdateCustomIntegration)
		// organisationUrl.GET("/:org_id/integrations/custom/:integration_id/settings", integrationsCtrl.GetCustomIntegrationSettings)
		// organisationUrl.GET("/:org_id/integrations/custom/:integration_id/status", integrationsCtrl.GetCustomIntegrationStatus)
		// organisationUrl.PUT("/:org_id/integrations/custom/:integration_id/settings", integrationsCtrl.UpdateCustomIntegrationSettings)

		// // Channel integrations routes
		// organisationUrl.GET("/:org_id/channels/:channel_id/integrations", integrationsCtrl.GetOrganisationChannelIntegrations)
		// organisationUrl.PATCH("/:org_id/channels/:channel_id/integrations/change-sendback-status", integrationsCtrl.ChangeOrgChannelIntSendBackStatus)
		// organisationUrl.POST("/:org_id/integrations/:integration_id/channels/:channel_id", integrationsCtrl.ActivateDeactivateChannelIntegration)
		// organisationUrl.GET("/:org_id/integrations/:integration_id/channels", integrationsCtrl.IntegrationChannels)
		// organisationUrl.GET("/:org_id/integrations/:integration_id/status", integrationsCtrl.CheckIntegrationIsActive)

		// // Organisation integration settings routes
		// organisationUrl.POST("/:org_id/integrations/:integration_id/settings", integrationsCtrl.AddIntegrationSetting)
		// organisationUrl.GET("/:org_id/integrations/:integration_id/settings", integrationsCtrl.GetIntegrationSettings)
		// organisationUrl.GET("/:org_id/integrations/:integration_id/integration-api-key", integrationsCtrl.GetIntegrationSettings)
		// organisationUrl.PATCH("/:org_id/integrations/:integration_id/settings/:setting_id", integrationsCtrl.UpdateIntegrationSetting)

		// // Organisation channel integration settings routes
		// organisationUrl.POST("/:org_id/integrations/:integration_id/channels/:channel_id/settings", integrationsCtrl.AddChannelIntegrationSetting)
		// organisationUrl.GET("/:org_id/integrations/:integration_id/channels/:channel_id/settings", integrationsCtrl.GetChannelIntegrationSetting)
		// organisationUrl.PATCH("/:org_id/integrations/:integration_id/channels/:channel_id/settings/:setting_id", integrationsCtrl.UpdateChannelIntegrationSetting)
		// organisationUrl.DELETE("/:org_id/integrations/:integration_id/channels/:channel_id/settings/:setting_id", integrationsCtrl.DeleteChannelIntegrationSetting)

		// // Integration slash commands routes
		// organisationUrl.POST("/:org_id/integrations/:integration_id/slash-commands", integrationsCtrl.AddIntegrationSlashCommand)
		// organisationUrl.GET("/:org_id/integrations/:integration_id/slash-commands", integrationsCtrl.GetIntegrationSlashCommands)
		// organisationUrl.GET("/:org_id/slash-commands", integrationsCtrl.GetAllOrgSlashCommands)
		// organisationUrl.PATCH("/:org_id/integrations/:integration_id/slash-commands/:command_id", integrationsCtrl.UpdateIntegrationSlashCommand)
		// organisationUrl.DELETE("/:org_id/integrations/:integration_id/slash-commands/:command_id", integrationsCtrl.DeleteIntegrationSlashCommand)

		// DM endpoints
		organisationUrl.POST("/:org_id/dms", dmCtrl.CreateDmChannel)
		organisationUrl.DELETE("/:org_id/dms/:channel_id", dmCtrl.DeleteDmChannel)
		organisationUrl.GET("/:org_id/dms", dmCtrl.GetDmChannels)
		organisationUrl.GET("/:org_id/dms/user/:user_id", dmCtrl.GetDmUser)

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
