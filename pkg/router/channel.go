package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Channels(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	channel := channel.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	channelUrl := r.Group(fmt.Sprintf("%v/channels", ApiVersion), middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql), middleware.MonitorFreeSub(db.Postgresql))
	channelQueueUrl := r.Group(fmt.Sprintf("%v/channels", ApiVersion))

	{
		// POST routes
		channelUrl.POST("", channel.CreateChannels)
		channelQueueUrl.POST("/backend-queue", channel.SaveIncomingQueueMsg)
		channelUrl.POST("/:channelId/messages", channel.AddChannelsMsg)
		channelUrl.POST("/:channelId/join", channel.JoinChannels)
		channelUrl.POST("/:channelId/leave", channel.LeaveChannels)
		channelUrl.POST("/add", channel.AddMembersToChannel)
		channelUrl.POST("/add-multiple", channel.AddMultipleMembersToChannel)
		channelUrl.POST("/:channelId/integration-channels", channel.AddIntegrationChannel)

		// PUT routes
		channelUrl.PUT("/:channelId/messages", channel.EditChannelsMsg)
		channelUrl.PUT("/:channelId/archive", channel.ArchiveChannel)

		// DELETE routes
		channelUrl.DELETE("/:channelId/messages/:messageId", channel.DeleteChannelsMsg)
		channelUrl.DELETE("/:channelId", channel.DeleteChannels)
		channelUrl.DELETE("/:channelId/integration-channels", channel.DeleteChannelIntegration)

		// GET routes
		channelUrl.GET("/:channelId/messages", channel.GetChannelsMsg)
		channelUrl.GET("/name/:channelName", channel.GetChannelsByName)
		channelUrl.GET("/:channelId", channel.GetChannel)
		channelUrl.GET("/:channelId/user-exist", channel.CheckUser)
		channelUrl.GET("/:channelId/num-users", channel.CountChannelsUsers)
		channelUrl.GET("/search/:channelName", channel.SearchChannelsByNames)
		channelUrl.GET("/:channelId/users", channel.GetUsersInChannel)
		channelUrl.GET("/:channelId/integration-channels/:IntModId", channel.GetIntegrationChannels)

		// PATCH routes
		channelUrl.PATCH("/:channelId/username", channel.UpdateUsername)
		channelUrl.PATCH("/:channelId", channel.UpdateChannels)
	}

	return r
}
