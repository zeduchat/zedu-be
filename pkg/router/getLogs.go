package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/logs"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func GetRecentLogs(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	logs := logs.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	logUrl := r.Group(fmt.Sprintf("%v", ApiVersion))

	{
		logUrl.GET("/logs", logs.GetTailLogs)
	}

	return r
}
