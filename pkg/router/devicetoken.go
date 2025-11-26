package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/pkg/controller/devicetoken"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// DeviceTokens wires the routes for registering device tokens.
func DeviceTokens(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	controller := devicetoken.Controller{Db: db, Validator: validator, Logger: logger}

	deviceRoutes := r.Group(fmt.Sprintf("%v/device-tokens", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		deviceRoutes.POST("", controller.Register)
	}

	return r
}
