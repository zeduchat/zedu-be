package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	admin "github.com/hngprojects/telex_be/services/admin"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateAdmin(c *gin.Context) {

	var (
		req models.CreateAdminRequest
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Failed to parse request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Error("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, err := admin.CreateAdmin(base.Db, req, c)
	if err != nil {
		base.Logger.Error("Failed to create group", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to add an admin", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("admin created successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Admin created successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) LoginAdmin(c *gin.Context) {
	var req models.AdminLoginRequest

	err := c.ShouldBind(&req)
	if err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := admin.LoginAdmin(req, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("Failed to login admin", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("admin login successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "Admin login successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ListAdmins(c *gin.Context) {

	respData, err := models.GetAllAdmins(base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(400, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("success")

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteAdmin(c *gin.Context) {
	admin_id := c.Param("admin_id")

	if _, err := uuid.Parse(admin_id); err != nil {
		base.Logger.Error("invalid admin id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid admin id format", "failed to decode admin id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := models.DeleteAdmin(base.Db.Postgresql, admin_id)
	if err != nil {
		rd := utility.BuildErrorResponse(400, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("admin deleted successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "Admin deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ListUsers(c *gin.Context) {

	users, paginationResponse, code, err := admin.ListUsers(base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("failed to list users", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("users retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "users retrieved successfully", users, paginationResponse)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetPlatformCreditsSummary(c *gin.Context) {
	metrics, err := models.GetPlatformCreditSummary(base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to fetch platform credit summary", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch platform credit summary", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Platform credit summary retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Platform credit summary retrieved successfully", metrics)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) InviteLeaderboard(c *gin.Context) {
	orgID := c.Query("org_id")
	var orgPtr *string
	if orgID != "" {
		if _, err := uuid.Parse(orgID); err != nil {
			base.Logger.Error("invalid org_id format", err)
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org_id format", "invalid org_id format", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		orgPtr = &orgID
	}

	limit := 10
	if c.Query("limit") != "" {
		l, err := strconv.Atoi(c.Query("limit"))
		if err != nil || l <= 0 {
			if err != nil {
				base.Logger.Error("invalid limit - parse error", err)
			} else {
				base.Logger.Error("invalid limit - non-positive", fmt.Errorf("limit must be a positive integer: %d", l))
			}
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid limit", "limit must be a positive integer", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		if l > 100 {
			base.Logger.Error("limit exceeds maximum allowed", fmt.Errorf("limit cannot exceed 100: %d", l))
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "limit cannot exceed 100", "limit cannot exceed 100", nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		limit = l
	}

	users, code, err := admin.ListUsersByInvites(base.Db.Postgresql, orgPtr, limit)
	if err != nil {
		base.Logger.Error("failed to build invite leaderboard", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("invite leaderboard retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "invite leaderboard retrieved successfully", users)
	c.JSON(http.StatusOK, rd)
}

// InitiateChangeAdminRole starts the two-step process for role elevation
func (base *Controller) InitiateChangeAdminRole(c *gin.Context) {
	targetAdminID := c.Param("admin_id")

	if _, err := uuid.Parse(targetAdminID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid admin id format", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.ChangeAdminRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// Validate confirmation flag for superadmin promotion
	if req.NewRole == models.RoleSuperAdmin && (req.ConfirmSuperAdmin == nil || !*req.ConfirmSuperAdmin) {
		rd := utility.BuildErrorResponse(http.StatusPreconditionRequired, "error", "Explicit confirmation required for superadmin promotion", nil, nil)
		c.JSON(http.StatusPreconditionRequired, rd)
		return
	}

	claims := c.MustGet("adminClaims").(jwt.MapClaims)
	requesterID := claims["admin_id"].(string)

	resp, err := admin.InitiateRoleChange(base.Db, targetAdminID, req.NewRole, requesterID, c.ClientIP())
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusAccepted, "Role change initiated. Use the token to confirm.", resp)
	c.JSON(http.StatusAccepted, rd)
}

// ConfirmChangeAdminRole finalizes the role change using the token
func (base *Controller) ConfirmChangeAdminRole(c *gin.Context) {
	var req models.ConfirmRoleChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims := c.MustGet("adminClaims").(jwt.MapClaims)
	requesterID := claims["admin_id"].(string)

	err := admin.ConfirmRoleChange(base.Db, base.Logger, req.ConfirmationToken, requesterID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Admin role updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

// GetRoleAuditHistory provides filterable logs for superadmins
func (base *Controller) GetRoleAuditHistory(c *gin.Context) {
	var logs []models.SuperadminRoleChangeAuditLog
	db := base.Db.Postgresql

	query := db.Model(&models.SuperadminRoleChangeAuditLog{}).Where("action = ?", "ROLE_CHANGE_CONFIRMED")

	// Filtering by User ID
	if targetID := c.Query("target_id"); targetID != "" {
		query = query.Where("target_id = ?", targetID)
	}

	// Filtering by Date (Format: YYYY-MM-DD)
	if date := c.Query("date"); date != "" {
		query = query.Where("DATE(created_at) = ?", date)
	}

	if err := query.Order("created_at desc").Find(&logs).Error; err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch logs", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Audit logs retrieved successfully", logs)
	c.JSON(http.StatusOK, rd)
}
