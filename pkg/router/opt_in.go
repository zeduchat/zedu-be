package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	optin "github.com/hngprojects/telex_be/pkg/controller/optIn"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func OptIn(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	optIn := optin.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	optInUrl := r.Group(fmt.Sprintf("%v", ApiVersion))
	{
		optInUrl.POST("/optin", optIn.CreateOptInRecord)
	}

	return r
}
