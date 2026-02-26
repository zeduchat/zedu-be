package organisation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetOrgRoles(c *gin.Context) {

	orgId := c.Param("org_id")

	respData, code, err := service.GetOrgRoles(base.Db.Postgresql, orgId, c)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Roles retrieved successfully", respData)

	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAOrgRole(c *gin.Context) {
	var (
		orgId  = c.Param("org_id")
		roleId = c.Param("role_id")
	)
	respData, code, err := service.GetAOrgRole(base.Db.Postgresql, orgId, roleId, c)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Role retrieved successfully", respData)

	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CreateOrgRole(c *gin.Context) {

	orgId := c.Param("org_id")

	var (
		req = models.OrgRole{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := service.CreateOrgRoles(req, orgId, base.Db.Postgresql, c)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("org role created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Org role created successfully", respData)

	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) DeleteOrgRole(c *gin.Context) {

	var (
		orgId  = c.Param("org_id")
		roleId = c.Param("role_id")
	)

	code, err := service.DeleteOrgRole(base.Db.Postgresql, orgId, roleId, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Role deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateOrgRole(c *gin.Context) {

	var (
		orgId  = c.Param("org_id")
		roleId = c.Param("role_id")
		req    = models.OrgRole{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := service.UpdateOrgRoles(req, orgId, roleId, base.Db.Postgresql, c)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("org role updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Org role updated successfully", respData)

	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateOrgPermissions(c *gin.Context) {

	var (
		orgId  = c.Param("org_id")
		roleId = c.Param("role_id")
		req    = models.Permission{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := service.UpdateOrgPermissions(req, orgId, roleId, base.Db.Postgresql, base.Db.Redis, c)

	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("permission updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Permissions updated successfully", nil)

	c.JSON(http.StatusOK, rd)
}
