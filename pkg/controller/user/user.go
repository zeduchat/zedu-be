package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"strings"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	profile "github.com/hngprojects/telex_be/services/profile"
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
		base.Logger.Error("failed to fetch user organisations", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user organisations retrieved successfully")
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

	userClaims := common.GetAllUserClaims(ctx)
	loggedUserID, ok := userClaims["user_id"].(string)
	if !ok {
		base.Logger.Error("failed to get user id from claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user id from claims", "failed to get user id from claims", nil)
		ctx.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := service.DeactiveUser(base.Db.Postgresql, userID, loggedUserID)
	if err != nil {
		base.Logger.Error("failed to deactivate user", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		ctx.JSON(code, rd)
		return
	}

	base.Logger.Info("user deactivated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User deactivated successfully", nil)
	ctx.JSON(http.StatusOK, rd)

}

// PatchUserStatus performs a partial, atomic update of a user's status.
func (base *Controller) PatchUserStatus(c *gin.Context) {
	userID := c.Param("user_id")

	if !utility.IsValidUUID(userID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", "invalid user id format", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := common.GetAllUserClaims(c)
	loggedUserID, ok := userClaims["user_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "unable to get user id from claims", "failed to get user id from claims", nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if loggedUserID != userID {
		rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "forbidden to update another user's status", "forbidden", nil)
		c.JSON(http.StatusForbidden, rd)
		return
	}

	var req models.PartialStatusUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if req.Text == nil && req.Emoji == nil && req.Expiry == nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "no fields provided to update", "no fields provided to update", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if code, err := validatePartialStatusInput(&req); err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	req.UserID = userID

	status, code, err := profile.PartialUpdateProfileStatus(req, base.Db.Postgresql, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "Failed to update user status", err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User status updated successfully", status)
	c.JSON(http.StatusOK, rd)
}

// GetUserStatus retrieves the current status for the authenticated user.
func (base *Controller) GetUserStatus(c *gin.Context) {
	userID := c.Param("user_id")

	if !utility.IsValidUUID(userID) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", "invalid user id format", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := common.GetAllUserClaims(c)
	loggedUserID, ok := userClaims["user_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "unable to get user id from claims", "failed to get user id from claims", nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if loggedUserID != userID {
		rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "forbidden to view another user's status", "forbidden", nil)
		c.JSON(http.StatusForbidden, rd)
		return
	}

	status, code, err := profile.GetUserStatus(userID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", "Failed to get user status", err, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "User status retrieved successfully", status)
	c.JSON(http.StatusOK, rd)
}

func validatePartialStatusInput(req *models.PartialStatusUpdate) (int, error) {
	if req.Text != nil && len(*req.Text) > 255 {
		return http.StatusBadRequest, errors.New("text must not exceed 255 characters")
	}

	if req.Emoji != nil {
		if len(*req.Emoji) > 64 {
			return http.StatusBadRequest, errors.New("emoji must not exceed 64 characters")
		}

		if strings.ContainsAny(*req.Emoji, " \t\n\r") {
			return http.StatusBadRequest, errors.New("emoji must not contain whitespace")
		}
	}

	if req.Expiry != nil && *req.Expiry < 0 {
		return http.StatusBadRequest, errors.New("expiry must be a non-negative unix timestamp (seconds)")
	}

	return 0, nil
}

func (base *Controller) SwitchUserOrg(c *gin.Context) {
	var (
		req = models.SwitchUserOrgReqeust{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		base.Logger.Error("failed to bind request", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if _, err := uuid.Parse(req.CurrentOrg); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := common.GetAllUserClaims(c)
	userId, ok := userClaims["user_id"].(string)
	if !ok {
		base.Logger.Error("failed to get user id from claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user id from claims", "failed to get user id from claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	accessTokenID, ok := userClaims["access_uuid"].(string)
	if !ok {
		base.Logger.Error("failed to get access token id from claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get access token id from claims", "failed to get access token id from claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := service.SwitchUserOrg(base.Db.Postgresql, c, req, userId, accessTokenID)
	if err != nil {
		base.Logger.Error("error switching user organisation", err)
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

	response, code, err := service.GetUserRoleInOrganisation(user_id, org_id, base.Db.Postgresql)
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
