package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/controller/apistatus"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// ApiStatusSecretHeader carries the monitoring secret on POST /api-status.
const ApiStatusSecretHeader = "X-Status-Secret"

func ApiStatus(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	apistatus := apistatus.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	apistatusUrl := r.Group(fmt.Sprintf("%v", ApiVersion))

	{
		apistatusUrl.POST("/api-status",
			middleware.RequireServiceSecret(ApiStatusSecretHeader, config.GetConfig().Server.ApiStatusSecret),
			apistatus.UpdateAPIStatus)
		apistatusUrl.GET("/api-status", apistatus.GetAPIStatus)
	}

	return r
}
