package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/integrations"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"

	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Organisation(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	organisation := organisation.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	channel := channel.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	integrations := integrations.Controller{Db:db, Validator: validator, Logger:logger, ExtReq: extReq}

	organisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))
	{

		organisationUrl.POST("", organisation.CreateOrganisation)
		organisationUrl.GET("/:org_id", organisation.GetOrganisation)
		organisationUrl.DELETE("/:org_id", organisation.DeleteOrganisation)
		organisationUrl.PUT("/:org_id", organisation.UpdateOrganisation)
		organisationUrl.POST("/:org_id/roles", organisation.CreateOrgRole)
		organisationUrl.GET("/:org_id/roles", organisation.GetOrgRoles)
		organisationUrl.GET("/:org_id/roles/:role_id", organisation.GetAOrgRole)
		organisationUrl.DELETE("/:org_id/roles/:role_id", organisation.DeleteOrgRole)
		organisationUrl.PUT("/:org_id/roles/:role_id", organisation.UpdateOrgRole)
		organisationUrl.PUT("/:org_id/roles/:role_id/permissions", organisation.UpdateOrgPermissions)
		organisationUrl.GET("/:org_id/channels", organisation.GetAllChannelssInOrganisation)

		//User Management Routes
		organisationUrl.GET("/:org_id/users", organisation.GetUsersInOrganisation)
		organisationUrl.GET("/:org_id/metrics", organisation.GetOrganisationCountMetrics)
		organisationUrl.DELETE("/:org_id/users/:user_id", organisation.RemoveMemberFromOrganisation)
		organisationUrl.PUT("/:org_id/users/:user_id", organisation.UpdateMember)
		organisationUrl.GET("/:org_id/invites", organisation.GetOrganisationInvites)
		organisationUrl.POST("/:org_id/users", organisation.AddMemberToOrganisation)

		organisationUrl.GET("/:org_id/user-channels", channel.GetUserChannels)
		organisationUrl.GET("/:org_id/user-not-channels", channel.GetUserNotInChannels)
		organisationUrl.PUT("/:org_id/archive-channel", channel.ArchiveChannel)

		//organisations integrations
		organisationUrl.POST("/:org_id/integrations", integrations.CreateIntegrationApp)
		organisationUrl.GET("/:org_id/integrations", integrations.GetAllIntegrationApp)
		organisationUrl.PATCH("/:org_id/integrations/:integration_id", integrations.UpdateIntegrationApp)
		organisationUrl.DELETE("/:org_id/integrations/:integration_id", integrations.DeleteIntegrationApp)
		organisationUrl.PATCH("/:org_id/integrations/:integration_id/active_status", integrations.SetIntegrationActiveStatus)

		//channels integrations
		organisationUrl.GET("/:org_id/channels/:channel_id/integrations", integrations.GetOrganisationChannelIntegrations)
		organisationUrl.POST("/:org_id/channels/:channel_id/integrations/:integration_id",integrations.ActivateChannelIntegration)
		organisationUrl.PATCH("/:org_id/channels/:channel_id/integrations/:integration_id",integrations.DeactivateChannelIntegration)
	}

	testOrganisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion))
	{
		testOrganisationUrl.GET("/:org_id/load-metrics", organisation.GetLoadingMetrics)
	}

	return r
}
