package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/savedmessages"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func SavedMessages(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	savedMessages := savedmessages.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	savedMessagesUrl := r.Group(fmt.Sprintf("%v/messages", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		savedMessagesUrl.POST("/save", savedMessages.SaveMessageForLater)
		savedMessagesUrl.GET("/saved", savedMessages.GetAllSavedMessages)
		savedMessagesUrl.DELETE("/message/:id", savedMessages.DeleteMessageByID)
	}

	return r
}
