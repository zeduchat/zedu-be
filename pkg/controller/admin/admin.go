package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	admin "github.com/hngprojects/telex_be/services/admin"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/audit_utility"
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

func (base *Controller) InitiateChangeAdminRole(c *gin.Context) {
	targetAdminID := c.Param("admin_id")

	if !utility.IsValidUUID(targetAdminID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid admin id format", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.ChangeAdminRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Error("Validation failed for role change", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if req.IsConfirmed == nil {
		base.Logger.Error("Admin role change attempted without confirmation flag", nil)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "confirm_superadmin is required when changing admin roles.", nil, nil)
		c.JSON(http.StatusPreconditionRequired, rd)
		return
	}
	if !*req.IsConfirmed {
		base.Logger.Error("Admin role change attempted with confirmation=false", nil)
		rd := utility.BuildErrorResponse(http.StatusConflict, "error", "You must explicitly confirm admin role change by setting confirm_superadmin to true.", nil, nil)
		c.JSON(http.StatusPreconditionRequired, rd)
		return
	}

	claims := c.MustGet("adminClaims").(jwt.MapClaims)
	requesterID := claims["admin_id"].(string)
	ipAddress := audit_utility.GetClientIP(c)

	resp, err := admin.InitiateAdminRoleChange(base.Db, targetAdminID, req.NewRole, requesterID, ipAddress)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusAccepted, "Role change initiated. Use the token to confirm.", resp)
	c.JSON(http.StatusAccepted, rd)
}

func (base *Controller) ConfirmChangeAdminRole(c *gin.Context) {
	var req models.ConfirmRoleChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Error("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims := c.MustGet("adminClaims").(jwt.MapClaims)
	requesterID := claims["admin_id"].(string)

	ipAddress := audit_utility.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" {
		userAgent = "Unknown/Direct API Request"
	}

	err := admin.ConfirmAdminRoleChange(base.Db, base.Logger, req.ConfirmationToken, requesterID, ipAddress, userAgent)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Admin role updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetRoleAuditHistory(c *gin.Context) {
	dateStr := c.Query("date")

	if dateStr != "" {
		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			base.Logger.Error("Invalid date format provided", err)
			rd := utility.BuildErrorResponse(
				http.StatusBadRequest,
				"error",
				"Invalid date format. Use YYYY-MM-DD",
				err.Error(),
				nil,
			)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
	}

	var auditModel models.AuditLog

	logs, pagination, err := auditModel.GetAuditHistory(base.Db.Postgresql, c)

	if err != nil {
		base.Logger.Error("Failed to retrieve audit history", err)
		rd := utility.BuildErrorResponse(
			http.StatusInternalServerError,
			"error",
			"Failed to fetch audit logs",
			err.Error(),
			nil,
		)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(
		http.StatusOK,
		"Audit logs retrieved successfully",
		map[string]any{
			"logs":       logs,
			"pagination": pagination,
		},
	)
	c.JSON(http.StatusOK, rd)
}
