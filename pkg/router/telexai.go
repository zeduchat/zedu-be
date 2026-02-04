package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/telexai"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func TelexAI(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	aiProxyCtrl := telexai.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	aiProxyUrl := r.Group(fmt.Sprintf("%v/telexai", ApiVersion), middleware.APIKeyAuthMiddleware(db.Postgresql, logger, false))
	{
		aiProxyUrl.POST("/restful/chat", aiProxyCtrl.RespondToChat)
		aiProxyUrl.GET("/models", aiProxyCtrl.ListModels)
		aiProxyUrl.GET("/models/tools", aiProxyCtrl.ListToolsModels)
	}

	chatCompletionsProxyUrl := r.Group(fmt.Sprintf("%v/telexai", ApiVersion), middleware.ProxyKeyAuthMiddleware(db.Postgresql, logger))
	{
		chatCompletionsProxyUrl.Any("/chat/completions", aiProxyCtrl.ProxyToOpenRouter)
		chatCompletionsProxyUrl.Any("/responses", aiProxyCtrl.ProxyToOpenRouter)
	}

	return r
}
