package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/newsletter"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Newsletter(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	newsletter := newsletter.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	newsletterUrl := r.Group(fmt.Sprintf("%v", ApiVersion))

	{
		newsletterUrl.POST("/newsletter", newsletter.SubscribeNewsLetter)
	}

	return r
}
