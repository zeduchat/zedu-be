package test_huddle

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/pkg/controller/huddle"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupHuddlesTestRouter() (*gin.Engine, *huddle.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	huddleController := &huddle.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	r := gin.Default()
	huddleGroup := r.Group("/api/v1/huddles", middleware.Authorize(huddleController.Db.Postgresql))
	{
		huddleGroup.PUT("/:id/camera", huddleController.UpdateCamera)
	}

	return r, huddleController
}
