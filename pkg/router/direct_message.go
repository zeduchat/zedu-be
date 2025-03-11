package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	dm "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Dms(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	dmCtrl := dm.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	thread := thread.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	threadUrl := r.Group(fmt.Sprintf("%v/dms", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		threadUrl.GET("/channels/:channel_id/threads", thread.GetAllChannelThreads)
		threadUrl.GET("/thread/:thread_id/channels/:channel_id", thread.GetUserSingleThreads)
		threadUrl.PUT("/thread/:thread_id/channels/:channel_id", thread.UpdateThreadMessage)
		threadUrl.DELETE("/thread/:thread_id/channels/:channel_id", dmCtrl.DeleteAThreadDm)
		threadUrl.POST("/channels/:channel_id/threads", dmCtrl.AddAThreadDm)

		threadUrl.POST("/messages/:channelId", dmCtrl.AddChannelsMsg)
		threadUrl.PUT("/messages/:channelId", dmCtrl.EditChannelsMsg)
		threadUrl.DELETE("/channels/:channelId/messages/:messageId", dmCtrl.DeleteChannelsMsg)
	}
	return r
}

