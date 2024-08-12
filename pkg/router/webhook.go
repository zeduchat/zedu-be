package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/webhook"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Router(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	webhook := webhook.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	webhookUrl := r.Group(fmt.Sprintf("%v/webhook", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		webhookUrl.GET("/:webhook_id", webhook.GetAWebhook)
		webhookUrl.POST("/:webhook_id", webhook.PostWebhook)

		webhookUrl.GET("/:webhook_id/history", webhook.GetWebhookHistory)
		webhookUrl.GET("/", webhook.GetAllWebhook)
		webhookUrl.POST("/", webhook.CreateWebhook)
		webhookUrl.DELETE("/:webhook_id", webhook.DeleteWebhook)
		webhookUrl.POST("/:webhook_id/change-status", webhook.ChangeWebhookStatus)

	}

	return r
}
