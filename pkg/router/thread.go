package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Threads(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	thread := thread.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	threadUrl := r.Group(fmt.Sprintf("%v/threads", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		threadUrl.GET("/organisations/:org_id", thread.GetAllUserOrgThreads)
		threadUrl.GET("/channels/:channel_id", thread.GetAllChannelThreads)
		threadUrl.GET("/:thread_id", thread.GetUserSingleThreads)
		threadUrl.PUT("/:thread_id", middleware.CheckIsDeactivated(db.Postgresql), thread.UpdateAThread)
		threadUrl.POST("", thread.AddAThread)
	}

	return r
}
