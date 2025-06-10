package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/pinnedmessages"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func PinMessages(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	pinnedMessages := pinnedmessages.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	pinnedMessagesUrl := r.Group(fmt.Sprintf("%v/channels", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		pinnedMessagesUrl.POST("/:channelId/pin", pinnedMessages.PinMessage)
		pinnedMessagesUrl.GET("/:channelId/pinned-messages", pinnedMessages.GetAllPinnedMessages)
		pinnedMessagesUrl.DELETE("/:channelId/pin/:messageId", pinnedMessages.UnPinMessage)
	}

	return r
}
