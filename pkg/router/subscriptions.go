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
	{
		subscriptionUrl.POST("/create", subscription.CreateSubscription)
		subscriptionUrl.GET("/list/:user_id", subscription.ListSubscriptions)
		subscriptionUrl.PUT("/modify", subscription.ModifySubscription)
		subscriptionUrl.DELETE("/delete", subscription.DeleteSubscription)
	}
	return r
}
