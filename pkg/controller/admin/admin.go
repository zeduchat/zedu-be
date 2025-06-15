package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
