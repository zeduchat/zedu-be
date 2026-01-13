package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/admin"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Admin(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	admin := admin.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	// Regular admin endpoints (all admins including super admins)
	adminAuthUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.AdminAuthorize(db.Postgresql))
	{
		adminAuthUrl.GET("/admins", admin.ListAdmins)
		adminAuthUrl.DELETE("/admins/:admin_id", admin.DeleteAdmin)
	}

	// Super admin only endpoints
	superAdminAuthUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.SuperAdminAuthorize(db.Postgresql))
	{
		superAdminAuthUrl.POST("/admins", admin.CreateAdmin)                                 // Only super admins can create admins
		superAdminAuthUrl.GET("/admins/users", admin.ListUsers)                              // Only super admins can list all users
		superAdminAuthUrl.GET("/dashboard/credits-summary", admin.GetPlatformCreditsSummary) // Platform credit metrics
		superAdminAuthUrl.GET("/admins/users/invites", admin.InviteLeaderboard)              // Super admins can view invite leaderboard
	}

	// Public admin endpoints (no authentication required)
	adminUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion))
	{
		adminUrl.POST("/login", admin.LoginAdmin)
	}

	return r
}
