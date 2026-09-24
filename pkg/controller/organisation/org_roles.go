package organisation

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/audit_utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func getOrgIDFromContext(c *gin.Context, db *storage.Database) string {
	if claimOrgID, err := middleware.GetUserClaims(c, db.Postgresql, "org_id"); err == nil {
		if idStr, ok := claimOrgID.(string); ok && idStr != "" && idStr != "00000000-0000-0000-0000-000000000000" {
			return idStr
		}
	}
	return c.Param("org_id")
}

func (base *Controller) GetOrgRoles(c *gin.Context) {

	orgId := getOrgIDFromContext(c, base.Db)

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
		orgId  = getOrgIDFromContext(c, base.Db)
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

	orgId := getOrgIDFromContext(c, base.Db)

	var (
		req = models.CreateOrgRoleRequest{}
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
	userClaims := common.GetAllUserClaims(c)
	actorID, _ := userClaims["user_id"].(string)
	var actorEmail string
	if actorID != "" {
		var user models.User
		if actor, err := user.GetUserByID(base.Db.Postgresql, actorID, orgId); err == nil {
			actorEmail = actor.Email
		}
	}
	if auditErr := audit_utility.CreateAuditLog(base.Db.Postgresql, audit_utility.AuditLogParams{
		ActorID:        actorID,
		ActorEmail:     actorEmail,
		ActorRole:      "user",
		OrganisationID: orgId,
		Action:         models.ActionRoleCreated,
		ResourceType:   models.ResourceRole,
		ResourceID:     req.Name,
		Description:    fmt.Sprintf("User %s created org role %s in organisation %s", actorEmail, req.Name, orgId),
		IPAddress:      audit_utility.GetClientIP(c),
		UserAgent:      c.GetHeader("User-Agent"),
		Success:        true,
	}); auditErr != nil {
		base.Logger.Error("failed to create audit log for org role creation: " + auditErr.Error())
	}
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Org role created successfully", respData)

	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) DeleteOrgRole(c *gin.Context) {

	var (
		orgId  = getOrgIDFromContext(c, base.Db)
		roleId = c.Param("role_id")
	)

	code, err := service.DeleteOrgRole(base.Db.Postgresql, base.Db.Redis, orgId, roleId, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	userClaims := common.GetAllUserClaims(c)
	actorID, _ := userClaims["user_id"].(string)
	var actorEmail string
	if actorID != "" {
		var user models.User
		if actor, err := user.GetUserByID(base.Db.Postgresql, actorID, orgId); err == nil {
			actorEmail = actor.Email
		}
	}
	if auditErr := audit_utility.CreateAuditLog(base.Db.Postgresql, audit_utility.AuditLogParams{
		ActorID:        actorID,
		ActorEmail:     actorEmail,
		ActorRole:      "user",
		OrganisationID: orgId,
		Action:         models.ActionRoleDeleted,
		ResourceType:   models.ResourceRole,
		ResourceID:     roleId,
		Description:    fmt.Sprintf("User %s deleted org role %s in organisation %s", actorEmail, roleId, orgId),
		IPAddress:      audit_utility.GetClientIP(c),
		UserAgent:      c.GetHeader("User-Agent"),
		Success:        true,
	}); auditErr != nil {
		base.Logger.Error("failed to create audit log for org role deletion: " + auditErr.Error())
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Role deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateOrgRole(c *gin.Context) {

	var (
		orgId  = getOrgIDFromContext(c, base.Db)
		roleId = c.Param("role_id")
		req    = models.UpdateOrgRoleRequest{}
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
	userClaims := common.GetAllUserClaims(c)
	actorID, _ := userClaims["user_id"].(string)
	var actorEmail string
	if actorID != "" {
		var user models.User
		if actor, err := user.GetUserByID(base.Db.Postgresql, actorID, orgId); err == nil {
			actorEmail = actor.Email
		}
	}
	if auditErr := audit_utility.CreateAuditLog(base.Db.Postgresql, audit_utility.AuditLogParams{
		ActorID:        actorID,
		ActorEmail:     actorEmail,
		ActorRole:      "user",
		OrganisationID: orgId,
		Action:         models.ActionRoleUpdated,
		ResourceType:   models.ResourceRole,
		ResourceID:     roleId,
		Description:    fmt.Sprintf("User %s updated org role %s in organisation %s", actorEmail, roleId, orgId),
		IPAddress:      audit_utility.GetClientIP(c),
		UserAgent:      c.GetHeader("User-Agent"),
		Success:        true,
	}); auditErr != nil {
		base.Logger.Error("failed to create audit log for org role update: " + auditErr.Error())
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Org role updated successfully", respData)

	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateOrgPermissions(c *gin.Context) {

	var (
		orgId  = getOrgIDFromContext(c, base.Db)
		roleId = c.Param("role_id")
		req    = models.UpdateOrgPermissionsRequest{}
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
	userClaims := common.GetAllUserClaims(c)
	actorID, _ := userClaims["user_id"].(string)
	var actorEmail string
	if actorID != "" {
		var user models.User
		if actor, err := user.GetUserByID(base.Db.Postgresql, actorID, orgId); err == nil {
			actorEmail = actor.Email
		}
	}
	if auditErr := audit_utility.CreateAuditLog(base.Db.Postgresql, audit_utility.AuditLogParams{
		ActorID:        actorID,
		ActorEmail:     actorEmail,
		ActorRole:      "user",
		OrganisationID: orgId,
		Action:         models.ActionPermissionsUpdated,
		ResourceType:   models.ResourceRole,
		ResourceID:     roleId,
		Description:    fmt.Sprintf("User %s updated permissions for role %s in organisation %s", actorEmail, roleId, orgId),
		IPAddress:      audit_utility.GetClientIP(c),
		UserAgent:      c.GetHeader("User-Agent"),
		Success:        true,
	}); auditErr != nil {
		base.Logger.Error("failed to create audit log for permissions update: " + auditErr.Error())
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Permissions updated successfully", nil)

	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetSystemPermissions(c *gin.Context) {
	permissions := models.GetMasterSystemPermissions()
	rd := utility.BuildSuccessResponse(http.StatusOK, "System permissions retrieved successfully", permissions)
	c.JSON(http.StatusOK, rd)
}
