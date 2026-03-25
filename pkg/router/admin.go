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

	// Regular admin endpoints
	adminAuthUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.AdminAuthorize(db.Postgresql))
	{
		adminAuthUrl.GET("/admins", admin.ListAdmins)
		adminAuthUrl.DELETE("/admins/:admin_id", admin.DeleteAdmin)
		adminAuthUrl.GET("/dashboard/credits-summary", admin.GetPlatformCreditsSummary)
		adminAuthUrl.GET("/dashboard/users-stats", admin.GetFreeVsPaidUserStats)
		adminAuthUrl.GET("/dashboard/ai-credits", admin.GetAICreditUsageStats)
		adminAuthUrl.GET("/dashboard/overview", admin.GetDashboardOverviewStats)
		adminAuthUrl.GET("/admins/users", admin.ListUsers)
		adminAuthUrl.GET("/admins/users/:user_id/details", admin.GetUserDetails)
		adminAuthUrl.GET("/admins/users/invites", admin.InviteLeaderboard)
		adminAuthUrl.GET("/plans", admin.GetPlans)
		adminAuthUrl.GET("/dashboard/user-growth", admin.GetUserGrowth)
		adminAuthUrl.GET("/admins/invitations/dashboard", admin.GetInvitationDashboard)
		adminAuthUrl.POST("/notifications/broadcast", admin.BroadcastNotification)
		adminAuthUrl.GET("/dashboard/activity", admin.GetAppActivity)

		// Billing management
		adminAuthUrl.GET("/billing/stats", admin.GetSubscriptionBillingStats)
		adminAuthUrl.GET("/billing/plans", admin.GetPlansFiltered)
		adminAuthUrl.POST("/billing/plans", admin.CreatePlan)
		adminAuthUrl.PUT("/billing/plans/:plan_id", admin.UpdatePlan)
		adminAuthUrl.DELETE("/billing/plans/:plan_id", admin.DeletePlan)

		// AI Credit packages management
		adminAuthUrl.GET("/billing/credit-packages/stats", admin.GetAICreditPackageStats)
		adminAuthUrl.GET("/billing/credit-packages", admin.GetAICreditPackagesFiltered)
		adminAuthUrl.POST("/billing/credit-packages", admin.CreateAICreditPackage)
		adminAuthUrl.PUT("/billing/credit-packages/:package_id", admin.UpdateAICreditPackage)
		adminAuthUrl.DELETE("/billing/credit-packages/:package_id", admin.DeleteAICreditPackage)

		// Admin transactions history
		adminAuthUrl.GET("/billing/transactions/stats", admin.GetAdminTransactionStats)
		adminAuthUrl.GET("/billing/transactions", admin.GetAdminTransactionsHistory)
	}

	// Super admin only endpoints
	superAdminAuthUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion), middleware.AdminAuthorize(db.Postgresql), middleware.RequireSuperAdmin())
	{
		superAdminAuthUrl.POST("/admins", admin.CreateAdmin) // Only super admins can create admins
		superAdminAuthUrl.POST("/admins/:admin_id/role/initiate", admin.InitiateChangeAdminRole)
		superAdminAuthUrl.POST("/admins/role/confirm", admin.ConfirmChangeAdminRole)
		superAdminAuthUrl.GET("/admins/audit-logs", admin.GetRoleAuditHistory)

	}

	// Public admin endpoints
	adminUrl := r.Group(fmt.Sprintf("%v/backoffice", ApiVersion))
	{
		adminUrl.POST("/login", admin.LoginAdmin)
	}

	return r
}
