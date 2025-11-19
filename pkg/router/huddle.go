package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/pkg/controller/huddle"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Huddles(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	ctrl := huddle.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	huddleGroup := r.Group(fmt.Sprintf("%v/huddles", ApiVersion), middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
	{
		huddleGroup.POST("/create", ctrl.Create)
		huddleGroup.POST("/:id/join", ctrl.Join)
		huddleGroup.PATCH("/:id/camera", ctrl.UpdateCamera)
	}

	return r
}
