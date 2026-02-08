package test_channel

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func SetupChannelTestRouter() (*gin.Engine, *channel.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tests.Setup()
	db := storage.Connection()
	validator := validator.New()
	utility.RegisterCustomValidations(validator)

	channelController := &channel.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupChannelRoutes(r, channelController)
	return r, channelController
}

func SetupChannelRoutes(r *gin.Engine, channelController *channel.Controller) {
	// Channel notification preference routes
	r.PUT("/api/v1/channels/:channelId/notification-preferences",
		middleware.Authorize(channelController.Db.Postgresql),
		channelController.UpdateDeviceNotification)

	r.GET("/api/v1/channels/:channelId/notification-preferences",
		middleware.Authorize(channelController.Db.Postgresql),
		channelController.GetChannelNotificationPref)

	r.GET("/api/v1/organisations/:org_id/notification-preferences/channels",
		middleware.Authorize(channelController.Db.Postgresql),
		channelController.GetUserChannelsNotificationPrefs)
}
