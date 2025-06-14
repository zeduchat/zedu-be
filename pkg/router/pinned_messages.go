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
	pinnedMessagesUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		pinnedMessagesUrl.POST("/:org_id/channels/:channel_id/pin", pinnedMessages.PinThreadMessage)
		pinnedMessagesUrl.POST("/:org_id/channels/:channel_id/pin/:messageId", pinnedMessages.PinReplyMessage)
		pinnedMessagesUrl.GET("/:org_id/channels/:channel_id/pinned-messages", pinnedMessages.GetAllPinnedMessages)
		pinnedMessagesUrl.DELETE("/:org_id/channels/:channel_id/pin/thread/:threadId", pinnedMessages.UnPinThreadMessage)
		pinnedMessagesUrl.DELETE("/:org_id/channels/:channel_id/pin/message/:messageId", pinnedMessages.UnPinReplyMessage)
	}

	return r
}
