package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/pkg/controller/onesignal"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func OneSignal(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	ctrl := onesignal.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	notificationGroup := r.Group(fmt.Sprintf("%v/notifications/onesignal", ApiVersion), middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
	{
		notificationGroup.POST("/send", ctrl.SendNotification)
	}

	return r
}
