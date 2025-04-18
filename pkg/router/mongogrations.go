package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/mongogrations"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/mongodb"
	"github.com/hngprojects/telex_be/utility"
)

func Mongogrations(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	mongogrations := mongogrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	store := mongodb.MongoStore{}

	baseUrl := fmt.Sprintf("%v/mongo-integrations/collections", ApiVersion)
	mongogrationsUrl := r.Group(baseUrl, middleware.APIKeyAuthMiddleware(db.Postgresql, logger, &store))
	// mongogrationsUrl := r.Group(baseUrl)

	{
		mongogrationsUrl.POST("", mongogrations.CreateCollection)
		mongogrationsUrl.GET("", mongogrations.ListCollections)
		mongogrationsUrl.DELETE("/:collection_name", mongogrations.DeleteCollection)

		mongogrationsUrl.POST("/:collection_name/documents", mongogrations.CreateEntry)
		mongogrationsUrl.GET("/:collection_name/documents", mongogrations.ReadEntries)
		mongogrationsUrl.PUT("/:collection_name/documents/:entry_id", mongogrations.UpdateEntry)
		mongogrationsUrl.DELETE("/:collection_name/documents/:entry_id", mongogrations.DeleteEntry)

	}

	return r
}
