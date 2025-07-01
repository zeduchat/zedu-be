package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/savedMessages"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func SavedMessages(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	savedMessages := savedMessages.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	savedMessagesUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		savedMessagesUrl.POST("/:org_id/thread/save", savedMessages.SaveThreadMessageForLater)
		savedMessagesUrl.POST("/:org_id/message/save", savedMessages.SaveReplyMessageForLater)
		savedMessagesUrl.GET("/:org_id/saved/message", savedMessages.GetAllSavedMessages)
		savedMessagesUrl.DELETE("/:org_id/saved/message/:smId", savedMessages.DeleteSavedMessageByID)
	}

	return r
}
