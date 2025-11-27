package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Buzz(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	ctrl := buzz.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	buzzGroup := r.Group(fmt.Sprintf("%v/buzz", ApiVersion), middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
	{
		buzzGroup.POST("/create", ctrl.Create)
		buzzGroup.POST("/:id/join", ctrl.Join)
		buzzGroup.POST("/token", ctrl.GetAgoraToken)
		buzzGroup.POST("/:id/notes", ctrl.CreateNote)
		buzzGroup.GET("/:id/notes", ctrl.GetNotes)
		buzzGroup.PATCH("/:id/notes/:note_id", ctrl.UpdateNote)
		buzzGroup.PATCH("/:id/camera", ctrl.UpdateCamera)
	}

	return r
}
