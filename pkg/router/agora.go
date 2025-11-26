package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/pkg/controller/agora"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// Agora routes provide RTC/RTM token generation for authenticated users.
func Agora(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	controller := &agora.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	group := r.Group(fmt.Sprintf("%v/agora", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		group.POST("/rtc-token", controller.GenerateRTC)
		group.POST("/rtm-token", controller.GenerateRTM)
		group.POST("/rte-token", controller.GenerateBoth)
	}

	return r
}
