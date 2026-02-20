package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func SlashCommands(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	agentsCtrl := agents.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	slashCommandsUrl := r.Group(fmt.Sprintf("%v/slash-commands", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		slashCommandsUrl.GET("", agentsCtrl.GetDefaultSlashCommands)
		slashCommandsUrl.POST("/process", agentsCtrl.ProcessSlashCommand)
	}

	return r
}
