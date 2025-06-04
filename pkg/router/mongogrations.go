package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/mongogrations"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Mongogrations(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	mongogrations := mongogrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	baseUrl := fmt.Sprintf("%v/agent_db/collections", ApiVersion)
	mongogrationsUrl := r.Group(baseUrl, middleware.APIKeyAuthMiddleware(db.Postgresql, logger, true))

	{
		mongogrationsUrl.POST("", mongogrations.CreateCollection)
		mongogrationsUrl.POST("/:collection_name/documents", mongogrations.CreateDocument)
		mongogrationsUrl.GET("/:collection_name/documents", mongogrations.GetAllDocuments)
		mongogrationsUrl.GET("/:collection_name/documents/:document_id", mongogrations.GetDocument)
		mongogrationsUrl.PUT("/:collection_name/documents/:document_id", mongogrations.UpdateDocument)
		mongogrationsUrl.DELETE("/:collection_name/documents/:document_id", mongogrations.DeleteDocument)
	}

	baseUrl1 := fmt.Sprintf("%v/organisation", ApiVersion)
	mongogrationsUrl1 := r.Group(baseUrl1, middleware.Authorize(db.Postgresql))
	{
		mongogrationsUrl1.GET("/:org_id/agents/:agent_id/get-api-key", mongogrations.FetchAPIKey)

	}

	return r
}
