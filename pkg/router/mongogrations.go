package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/mongogrations"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Mongogrations(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	mongogrations := mongogrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	mongogrationsUrl := r.Group(fmt.Sprintf("%v", ApiVersion))
	{
		mongogrationsUrl.GET("/mongo-integrations", mongogrations.ReadEntries)
		mongogrationsUrl.POST("/mongo-integrations", mongogrations.CreateEntry)
		mongogrationsUrl.PUT("/mongo-integrations", mongogrations.UpdateEntry)
		mongogrationsUrl.DELETE("/mongo-integrations", mongogrations.DeleteEntry)
	}
	return r
}
