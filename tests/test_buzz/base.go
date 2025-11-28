package test_buzz

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupBuzzTestRouter() (*gin.Engine, *buzz.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	buzzController := &buzz.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	r := gin.Default()
	buzzGroup := r.Group("/api/v1/buzz", middleware.Authorize(buzzController.Db.Postgresql))
	{
		buzzGroup.PUT("/:id/camera", buzzController.UpdateCamera)
		buzzGroup.POST("/:id/notes", buzzController.CreateNote)
		buzzGroup.GET("/:id/notes", buzzController.GetNotes)
		buzzGroup.PATCH("/:id/notes/:note_id", buzzController.UpdateNote)
		buzzGroup.POST("/create", buzzController.Create)
		buzzGroup.POST("/:id/leave", buzzController.LeaveBuzz)
	}

	return r, buzzController
}
