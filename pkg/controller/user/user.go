package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetAllUsers(c *gin.Context) {

	usersData, paginationResponse, code, err := service.GetAllUsers(c, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Users retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAUser(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	userData, code, err := service.GetAUser(userID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetOrganisationDetails(c *gin.Context) {
	var (
		org_id = c.Param("org_id")
	)

	userData, code, err := service.GetOrganisationDetails(base.Db.Postgresql, c, org_id)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation details retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAUserOrganisation(c *gin.Context) {

	userData, code, err := service.GetAUserOrganisation(base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User organisations retrieved successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteAUser(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	code, err := service.DeleteAUser(userID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateAUser(c *gin.Context) {
	var (
		userID = c.Param("user_id")
		req    = models.UpdateUserRequestModel{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := service.UpdateAUser(req, userID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user info updated successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "User info updated successfully", respData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) ActivateUser(ctx *gin.Context) {
	var (
		userID = ctx.Param("user_id")
	)

	code, err := service.ActivateUser(userID, base.Db.Postgresql, ctx)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		ctx.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User activated successfully", nil)
	ctx.JSON(http.StatusOK, rd)

}

func (base *Controller) DeactiveUser(ctx *gin.Context) {
	var (
		userID = ctx.Param("user_id")
	)

	code, err := service.DeactiveUser(userID, base.Db.Postgresql, ctx)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		ctx.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User deactivated successfully", nil)
	ctx.JSON(http.StatusOK, rd)

}

func (base *Controller) SwitchUserOrg(c *gin.Context) {
	var (
		req = models.SwitchUserOrgReqeust{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if _, err := uuid.Parse(req.CurrentOrg); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)

	userId := userClaims["user_id"].(string)

	respData, code, err := service.SwitchUserOrg(req, userId, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user org switched successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "User organisation switched successfully", respData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetUserRoleInOrganisation(c *gin.Context) {
	var (
		user_id = c.Param("user_id")
		org_id  = c.Param("org_id")
	)

	valid := utility.IsValidUUID(user_id)
	if !valid {
		base.Logger.Error("invalid user id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", "failed to decode user id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	valid = utility.IsValidUUID(org_id)
	if !valid {
		base.Logger.Error("invalid organisation id format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, code ,err := service.GetUserRoleInOrganisation(user_id, org_id, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("failed to fetch user role", err)
		rd := utility.BuildErrorResponse(code, "error", "failed to fetch user role", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user role fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User role fetched successfully", response)
	c.JSON(http.StatusOK, rd)
}
