package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/subscriptions"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Subscriptions(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	subscription := subscriptions.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	subscriptionUrl := r.Group(fmt.Sprintf("%v/subscriptions", ApiVersion), middleware.Authorize(db.Postgresql))
	webhookUrl := r.Group(fmt.Sprintf("%v/subscriptions", ApiVersion))
	{
		subscriptionUrl.POST("/create/", subscription.CreateSubscription)
		subscriptionUrl.GET("/list/:org_id", subscription.GetCurrentSubscription)
		subscriptionUrl.PUT("/modify/", subscription.ModifySubscription)
		subscriptionUrl.DELETE("/:org_id", subscription.DeleteSubscription)
		subscriptionUrl.POST("/complete/", subscription.CompleteSubscription)
		subscriptionUrl.GET("/invoice/download/:session_id/:org_id", subscription.DownloadInvoice)
		webhookUrl.POST("webhook", subscription.HandleStripeWebhook)
	}
	return r
}
