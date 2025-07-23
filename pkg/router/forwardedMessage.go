package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/forwardedMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func ForwardMessage(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	forwardedMessage := forwardedMessage.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	forwardedMessageUrl := r.Group(fmt.Sprintf("%v/channels", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		forwardedMessageUrl.POST("/:channelId/thread/forward", forwardedMessage.ForwardThreadMessage)
		forwardedMessageUrl.POST("/:channelId/message/forward", forwardedMessage.ForwardReplyMessage)
	}

	return r
}