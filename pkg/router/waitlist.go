package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/waitlist"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Waitlist(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	waitlist := waitlist.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	waitlistletterUrl := r.Group(fmt.Sprintf("%v", ApiVersion))

	{
		waitlistletterUrl.POST("/waitlist", waitlist.SubscribeWaitListLetter)
	}

	return r
}
