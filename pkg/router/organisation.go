package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
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
		organisationUrl.GET("/:org_id/users", organisationCtrl.GetUsersBotsInOrganisation)
		organisationUrl.GET("/:org_id/users/search", organisationCtrl.SearchUsersInOrganisation)
		organisationUrl.POST("/:org_id/users", organisationCtrl.AddMemberToOrganisation)
		organisationUrl.PUT("/:org_id/users/:user_id", organisationCtrl.UpdateMember)
		organisationUrl.PATCH("/:org_id/users/:user_id/status", organisationCtrl.ChangeMemberActiveStatus)
		organisationUrl.DELETE("/:org_id/users/:user_id", organisationCtrl.RemoveMemberFromOrganisation)
		organisationUrl.PATCH("/:org_id/users/:user_id/role", organisationCtrl.UpdateMemberRole)

		organisationUrl.GET("/:org_id/metrics", organisationCtrl.GetOrganisationCountMetrics)
		organisationUrl.GET("/:org_id/invites", organisationCtrl.GetOrganisationInvites)
		organisationUrl.GET("/:org_id/notification-preference", organisationCtrl.GetChannelNotificationPref)
		organisationUrl.POST("/:org_id/notification-preference", organisationCtrl.UpdateDeviceNotification)

		//bots
		organisationUrl.GET("/:org_id/fetch-bots", integrationsCtrl.FetchActiveOrganisationAgents)
		organisationUrl.GET("/:org_id/get-started", organisationCtrl.GetStarted)

		//Channels notification prefence
		organisationUrl.GET("/:org_id/channels/notification-preference", channelCtrl.GetUserChannelsNotificationPrefs)

		// User pinned organisations routes
		organisationUrl.POST("/pin", organisationCtrl.CreateUserPinnedOrganisation)
		organisationUrl.GET("/pin", organisationCtrl.GetUserPinnedOrganisations)
		organisationUrl.DELETE("/pin/:org_id", organisationCtrl.UnpinOrganisation)
	}

	// Test routes
	testOrganisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion))
	{
		testOrganisationUrl.GET("/:org_id/load-org-info", organisationCtrl.GetLoadingMetrics)
	}

	adminOrgUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.AdminAuthorize(db.Postgresql))
	{
		adminOrgUrl.GET("/organisations/all", organisationCtrl.GetAllOrganisations)
	}

	return r
}
