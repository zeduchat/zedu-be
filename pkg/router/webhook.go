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

func Webhook(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	webhook := webhook.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	webhookUrl := r.Group(fmt.Sprintf("%v/webhooks", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		webhookUrl.GET("/channel/:webhook_id", webhook.GetWebhook)
		webhookUrl.POST("/channel/:webhook_id", webhook.PostWebhook)

		webhookUrl.GET("/:channel_id/history/:webhook_id", webhook.GetWebhookHistory)
		webhookUrl.GET("/:channel_id", webhook.GetAllWebhook)
		webhookUrl.POST("/:channel_id", webhook.CreateWebhook)
		webhookUrl.DELETE("/:channel_id/:webhook_id", webhook.DeleteWebhook)
		webhookUrl.PUT("/:channel_id/:webhook_id", webhook.UpdateWebhook)
		webhookUrl.PUT("/:channel_id/:webhook_id/change-status", webhook.ChangeWebhookStatus)

	}

	return r
}
