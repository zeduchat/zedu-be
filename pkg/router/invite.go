package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/invitation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Invite(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	invite := invitation.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	inviteUrl := r.Group(fmt.Sprintf("%v/invite", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		inviteUrl.POST("/", invite.OrganisationCreateInvite)
		inviteUrl.GET("/:token", invite.OrganisationAcceptInvite)
	}
	return r
}
