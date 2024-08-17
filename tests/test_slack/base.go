package test_slack

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/slack"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupSlackTelexTestRouter() (*gin.Engine, *slack.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	slackController := &slack.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupSlackTelexRoutes(r, slackController)
	return r, slackController
}

func SetupSlackTelexRoutes(r *gin.Engine, slackController *slack.Controller) {
	slackUrl := r.Group("/api/v1", middleware.Authorize(slackController.Db.Postgresql))

	slackUrl.POST("/slack/access-token", slackController.SlackOauth)
	slackUrl.GET("/slack/access-token", slackController.GetSlackAccessToken)
	slackUrl.GET("/slack/channels", slackController.GetSlackChannels)

}
