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
	"github.com/hngprojects/telex_be/services/group"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) AddUser(c *gin.Context) {

	var (
		req    models.CreateGroupRequest
		org_id = c.Param("org_id")
	)

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

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

	req.OrganisationID = org_id

	response, err := group.CreateGroup(base.Db, req)
	if err != nil {
		base.Logger.Error("Failed to create group", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to create group", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group created successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group created successfully", response)
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

	respData, code, err := admin.LoginAdmin(req, base.Db.Postgresql, c, base.ExtReq)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("admin login successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "Admin login successfully", respData)
	c.JSON(http.StatusOK, rd)
}
