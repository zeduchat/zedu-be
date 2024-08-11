package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/teams"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Team(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	team := teams.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	teamUrl := r.Group(fmt.Sprintf("%v/teams", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		teamUrl.POST("/", team.CreateTeam)
		teamUrl.GET("/:teamId", team.GetTeamByID)
		teamUrl.GET("/rooms/:teamId", team.GetAllRoomsInTeam)
		teamUrl.DELETE("/:teamId", team.DeleteTeam)
	}
	return r
}
