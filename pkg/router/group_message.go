package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	dm "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func GroupDMs(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	gpdmCtrl := dm.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	thread := thread.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	channel := channel.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	dmCtrl := dm.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	threadUrl := r.Group(fmt.Sprintf("%v/group-dms", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		threadUrl.GET("/channels/:channel_id/threads", thread.GetAllChannelThreads)
		threadUrl.GET("/thread/:thread_id/channels/:channel_id", thread.GetUserSingleThreads)
		threadUrl.PUT("/thread/:thread_id/channels/:channel_id", thread.UpdateThreadMessage)
		threadUrl.DELETE("/thread/:thread_id/channels/:channel_id", thread.DeleteAThread)

		threadUrl.POST("/channels/:channel_id/threads", gpdmCtrl.AddAGroupThreadDM)
		threadUrl.POST("/messages/:channelId", gpdmCtrl.ReplyGroupChannelsMsg)
		threadUrl.PUT("/messages/:channelId", gpdmCtrl.EditGroupChannelsMsg)
		threadUrl.DELETE("/channels/:channelId/messages/:messageId", channel.DeleteChannelsMsg)
	}

	organisationUrls := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		// Group DM endpoints
		organisationUrls.POST("/:org_id/group-dms", dmCtrl.CreateGroupDMChannel)
		organisationUrls.POST("/group-dms/:channel_id/join", dmCtrl.JoinGroupDMChannel)
		organisationUrls.POST("/group-dms/:channel_id/participants", dmCtrl.AddParticipantsToGroupDMChannel)
		organisationUrls.DELETE("/:org_id/group-dms/:channel_id", dmCtrl.LeaveGroupDMChannel)
		organisationUrls.GET("/:org_id/group-dms", dmCtrl.GetGroupDMChannels)
		organisationUrls.GET("/:org_id/group-dms/:user_id", dmCtrl.GetUserGroupDMs)
	}
	return r
}
