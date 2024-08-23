package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/webhook"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func WebhookHandler(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	webhook := webhook.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	webhookFormerUrl := r.Group(fmt.Sprintf("%v/webhooks", ApiVersion))

	{
		webhookFormerUrl.POST("/feed/backend-queue", webhook.PostFeedWebhook)
	}

	webhookUrl := r.Group(fmt.Sprintf("%v/webhooks", "v1"))

	{
		webhookUrl.POST("/backend-queue", webhook.PostFeedWebhook)
	}

	return r
}
