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
		channelUrl.POST("", channel.CreateChannels)
		channelQueueUrl.POST("/backend-queue", channel.SaveIncomingQueueMsg)
		channelUrl.POST("/:channelId/messages", channel.AddChannelsMsg)
		channelUrl.PUT("/:channelId/messages", channel.EditChannelsMsg)
		channelUrl.GET("/:channelId/messages", channel.GetChannelsMsg)
		channelUrl.GET("/name/:channelName", channel.GetChannelsByName)
		channelUrl.POST("/:channelId/join", channel.JoinChannels)
		channelUrl.POST("/:channelId/leave", channel.LeaveChannels)
		channelUrl.DELETE("/:channelId", channel.DeleteChannels)
		channelUrl.PATCH("/:channelId/username", channel.UpdateUsername)
		channelUrl.GET("/:channelId", channel.GetChannels)
		channelUrl.GET("/:channelId/user-exist", channel.CheckUser)
		channelUrl.GET("/:channelId/num-users", channel.CountChannelsUsers)
		channelUrl.PATCH("/:channelId", channel.UpdateChannels)

		channelUrl.GET("/search/:channelName", channel.SearchChannelsByNames)
		channelUrl.GET("/:channelId/users", channel.GetUsersInChannel)
		channelUrl.POST("/add", channel.AddMembersToChannel)

		channelUrl.POST("/add-multiple", channel.AddMultipleMembersToChannel)

		channelUrl.PUT("/:channelId/archive", channel.ArchiveChannel)
	}

	return r
}
