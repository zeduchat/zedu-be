package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/shares"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Shares(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	controller := shares.Controller{Db: db, Logger: logger, Validator: validator, ExtReq: extReq}

	sharesURL := r.Group(fmt.Sprintf("%v/shares", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		sharesURL.POST("/", controller.Create)
		sharesURL.GET("/", controller.GetMyShares)
		sharesURL.GET("/performance", controller.GetPerformance)
		sharesURL.GET("/:id", controller.GetShare)
		sharesURL.DELETE("/:id", controller.Delete)
	}

	return r
}
