package test_activity

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/activity"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
)

func SetupActivityTestRouter() (*gin.Engine, *activity.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tests.Setup()
	db := storage.Connection()
	validatorRef := validator.New()

	activityController := &activity.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupActivityRoutes(r, activityController)
	return r, activityController
}

func SetupActivityRoutes(r *gin.Engine, activityController *activity.Controller) {
	r.GET("/api/v1/organisations/:org_id/mentions", middleware.Authorize(activityController.Db.Postgresql),
		activityController.GetUserMentionsActivity)
}
