package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/group"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Group(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	group := group.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	organisationUrl := r.Group(fmt.Sprintf("%v/organisations", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		//groups
		organisationUrl.GET("/:org_id/groups", group.GetGroups)
		organisationUrl.GET("/:org_id/groups/:group_id", group.GetGroup)
		organisationUrl.POST("/:org_id/groups", group.CreateGroup)
		organisationUrl.PATCH("/:org_id/groups/:group_id", group.UpdateGroup)
		organisationUrl.DELETE("/:org_id/groups/:group_id", group.DeleteGroup)

		//channels
		organisationUrl.GET("/:org_id/groups/:group_id/channels", group.GetGroupChannels)
		organisationUrl.GET("/:org_id/groups/channels/no-group", group.GetChannelsNotInGroup)
		organisationUrl.POST("/:org_id/groups/:group_id/channels", group.AssignGroupChannel)
		organisationUrl.DELETE("/:org_id/groups/:group_id/channels", group.RemoveGroupChannel)
		organisationUrl.PATCH("/:org_id/groups/:group_id/channels/:channel_id", group.MoveGroupChannel)
	}

	return r
}
