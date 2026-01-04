package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Threads(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	thread := thread.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	channel := channel.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	channelUrl := r.Group(fmt.Sprintf("%v/channels", ApiVersion), middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql), middleware.MonitorFreeSub(db.Postgresql))
	threadUrl := r.Group(fmt.Sprintf("%v/threads", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		threadUrl.GET("/recent", thread.GetUserRecentThreads)
		threadUrl.GET("/organisations/:org_id", thread.GetAllUserOrgThreads)
		threadUrl.GET("/channels/:channel_id/threads", thread.GetChannelThreads)
		threadUrl.GET("/channels/:channel_id", thread.GetAllChannelThreads)
		threadUrl.GET("/:thread_id/channels/:channel_id", thread.GetUserSingleThreads)
		threadUrl.PUT("/:thread_id/channels/:channel_id", thread.UpdateThreadMessage)
		threadUrl.DELETE("/:thread_id/channels/:channel_id", thread.DeleteAThread)

		// main channel thread
		threadUrl.POST("/:channel_id", thread.AddAThread)                   //add thread message
		channelUrl.POST("/:channelId/messages", channel.ReplyThreadMessage) //reply thread message

		threadUrl.GET("/channels/:channel_id/:searching", thread.SearchChannel)
		threadUrl.GET("/organisations/:org_id/metrics", thread.GetChannelCountInfo)
	}

	return r
}
