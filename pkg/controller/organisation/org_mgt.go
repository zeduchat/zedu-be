package organisation

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetOrganisationCountMetrics(c *gin.Context) {
	orgId := c.Param("org_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to delete organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	countMetricsData, err := organisation.CountMetrics(base.Db.Postgresql, userId, orgId)
	if err != nil {
		base.Logger.Error("failed to fetch metrics", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch metrics", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", countMetricsData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) RemoveMemberFromOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")
	userId := c.Param("user_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to remove member", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	ownerId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(userId); err != nil {
		base.Logger.Error("invalid user id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", "failed to decode user id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := organisation.RemoveMemberFromOrganisation(ownerId, orgId, userId, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("failed to remove member", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to remove member", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", "member removed successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateMember(c *gin.Context) {
	orgId := c.Param("org_id")
	userId := c.Param("user_id")

	var updateMemberRequest models.UpdateMemberRequest

	if err := c.ShouldBindJSON(&updateMemberRequest); err != nil {
		base.Logger.Error("failed to bind request", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to bind request", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to update member", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	ownerId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(userId); err != nil {
		base.Logger.Error("invalid user id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", "failed to decode user id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	resp, err := organisation.UpdateMember(base.Db.Postgresql, ownerId, orgId, userId, updateMemberRequest)
	if err != nil {
		base.Logger.Error("failed to update role", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to update role", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", resp)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetOrganisationInvites(c *gin.Context) {
	orgId := c.Param("org_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to get organisation invites", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	invitations, paginationResponse, err := organisation.GetOrganisationInvites(c, base.Db.Postgresql, userId, orgId)
	if err != nil {
		base.Logger.Error("failed to fetch organisation invites", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch organisation invites", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", invitations, paginationResponse)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) AddMemberToOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")

	var createOGMT models.OrgUserCreateRequest

	if err := c.ShouldBindJSON(&createOGMT); err != nil {
		base.Logger.Error("failed to bind request", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to bind request", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(createOGMT)
	if err != nil {
		base.Logger.Error("validation error", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "validation error", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(createOGMT.RoleID); err != nil {
		base.Logger.Error("invalid role id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid role id format", "failed to decode role id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to add member", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	ownerId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(createOGMT.UserID); err != nil {
		base.Logger.Error("invalid user id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", "failed to decode user id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = organisation.AddMemberToOrganisation(ownerId, orgId, createOGMT, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("failed to add member", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to add member", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", "member added successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateDeviceNotification(c *gin.Context) {
	var (
		req models.DeviceNotificationSettings
	)
	err := c.ShouldBindJSON(&req)
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

	req.OrgID = c.Param("org_id")

	if _, err := uuid.Parse(req.OrgID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	req.UserID = userClaims["user_id"].(string)

	respData, code, err := organisation.UpdateDeviceNotification(base.Db.Postgresql, base.Logger, req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("notification settings updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "notification settings updated successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) GetChannelNotificationPref(c *gin.Context) {

	orgId := c.Param("org_id")

	if _, err := uuid.Parse(orgId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	UserId := userClaims["user_id"].(string)

	DeviceType := c.Query("device_type")

	if valid, msg := ValidateDeviceType(DeviceType); !valid {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", msg, errors.New(msg), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":      orgId,
		"user_id":     UserId,
		"device_type": DeviceType,
	}

	respData, err := organisation.GetOrCreateDeviceNotification(base.Db.Postgresql, base.Logger, ids)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("device notification fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "device notification fetched successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func ValidateDeviceType(device string) (bool, string) {

	devices := map[string]bool{
		"web":     true,
		"desktop": true,
		"mobile":  true,
	}

	if device == "" {
		return false, "empty device type passed"
	}

	exist := devices[device]

	if !exist {
		return false, "invalid device passed, device supported: web, desktop, mobile"
	}

	return true, ""

}
