package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/slack"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Slack(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	slack := slack.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	slackUrl := r.Group(fmt.Sprintf("%v", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		slackUrl.POST("/slack/organisations/:orgId/token-exchg", slack.SlackOauth)
		slackUrl.GET("/slack/organisations/:orgId", slack.GetSlackAccessToken)
		slackUrl.GET("/slack/channels", slack.GetSlackChannels)
		slackUrl.POST("/organisation/{org_id}/channels/mapping", slack.CreateTelexSlackChannelMapping)


		slackUrl.GET("/slack/get-manifest", slack.GetManifest)
	}

	return r
}
