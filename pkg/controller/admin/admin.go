package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := admin.LoginAdmin(req, base.Db.Postgresql, c)
	if err != nil {
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
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

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
	users, paginationResponse, code, err := admin.ListUsersByInvites(base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "invite leaderboard retrieved successfully", users, paginationResponse)
	c.JSON(http.StatusOK, rd)
}
