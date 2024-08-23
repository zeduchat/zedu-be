package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/apistatus"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func ApiStatus(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	apistatus := apistatus.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	apistatusUrl := r.Group(fmt.Sprintf("%v", ApiVersion))

	{
		apistatusUrl.POST("/api-status", apistatus.UpdateAPIStatus)
		apistatusUrl.GET("/api-status", apistatus.GetAPIStatus)
	}

	return r
}
