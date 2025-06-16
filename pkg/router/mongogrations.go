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
	agentDB := mongogrations.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	baseUrl := fmt.Sprintf("%v/agent_db/collections", ApiVersion)
	agentDBUrl := r.Group(baseUrl, middleware.APIKeyAuthMiddleware(db.Postgresql, logger, true))

	{
		agentDBUrl.POST("", agentDB.CreateCollection)
		agentDBUrl.POST("/:collection_name/documents", agentDB.CreateDocument)
		agentDBUrl.GET("/:collection_name/documents", agentDB.GetAllDocuments)
		agentDBUrl.GET("/:collection_name/documents/:document_id", agentDB.GetDocument)
		agentDBUrl.PUT("/:collection_name/documents/:document_id", agentDB.UpdateDocument)
		agentDBUrl.DELETE("/:collection_name/documents/:document_id", agentDB.DeleteDocument)
	}

	baseUrl1 := fmt.Sprintf("%v/organisation", ApiVersion)
	agentDBUrl1 := r.Group(baseUrl1, middleware.Authorize(db.Postgresql))
	{
		agentDBUrl1.GET("/:org_id/agents/:agent_id/get-api-key", agentDB.FetchAPIKey)

	}

	return r
}
