package group

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/group"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateGroup(c *gin.Context) {

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

func (base *Controller) GetGroup(c *gin.Context) {
	var (
		org_id   = c.Param("org_id")
		group_id = c.Param("group_id")
	)

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"group_id":        group_id,
	}

	groups, err := group.GetGroup(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to get group", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to get group", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Groups fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Groups fetched successfully", groups)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetGroups(c *gin.Context) {
	var (
		org_id = c.Param("org_id")
	)

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"user_id":         userId,
	}

	groups, err := group.GetGroups(base.Db, ids)
	if err != nil {
		base.Logger.Error(err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to get groups", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Groups fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Groups fetched successfully", groups)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateGroup(c *gin.Context) {
	var (
		req models.UpdateGroupRequest
		ids map[string]string = map[string]string{
			"organisation_id": c.Param("org_id"),
			"group_id":        c.Param("group_id"),
		}
	)

	if _, err := uuid.Parse(ids["organisation_id"]); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(ids["group_id"]); err != nil {
		base.Logger.Error("Invalid group id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid group id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Error("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, err := group.UpdateGroup(base.Db, req, ids)
	if err != nil {
		base.Logger.Error("Failed to update group", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to update group", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group updated successfully", response)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) DeleteGroup(c *gin.Context) {
	var (
		org_id   = c.Param("org_id")
		group_id = c.Param("group_id")
	)

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("Invalid organisation id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid organisation id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(group_id); err != nil {
		base.Logger.Error("Invalid group id", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid request", "Invalid group id", err, nil)
		c.JSON(http.StatusBadRequest, rd)
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"group_id":        group_id,
	}

	err := group.DeleteGroup(base.Db, ids)
	if err != nil {
		base.Logger.Error("Failed to delete group", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Error", "Failed to delete group", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Group deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Group deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}
