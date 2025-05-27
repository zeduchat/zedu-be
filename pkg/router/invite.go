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

	basePath := fmt.Sprintf("%v/invite", ApiVersion)

	r.POST(
		basePath,
		middleware.Authorize(db.Postgresql),
		middleware.PlanMiddleware(db.Postgresql, db.Redis),
		invite.OrganisationCreateInvite,
	)

	inviteUrl := r.Group(basePath)
	{
		inviteUrl.POST("/verify", invite.OrganisationVerifyInvite)
		inviteUrl.POST("/resend", middleware.Authorize(db.Postgresql), invite.ResendInvitation)

		inviteUrl.POST("/general", middleware.Authorize(db.Postgresql), invite.GeneralInvitationCreate)
		inviteUrl.POST("/general/verify", middleware.Authorize(db.Postgresql), invite.GeneralInvitationVerify)
		inviteUrl.POST("/general/change-status", middleware.Authorize(db.Postgresql), invite.ChangeGeneralInviteStatus)

		inviteUrl.POST("/admin-resend", invite.AdminResend)
		inviteUrl.DELETE("/:invite_id", middleware.Authorize(db.Postgresql), invite.CancelInvitation)
	}

	return r
}
