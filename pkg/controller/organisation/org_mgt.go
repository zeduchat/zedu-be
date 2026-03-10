package organisation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/audit_utility"
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

	// Audit logging for user leaving organisation
	auditData := map[string]interface{}{
		"organisation_id": orgId,
		"user_id":         userId,
		"removed_by":      ownerId,
	}
	auditDataJSON, _ := json.Marshal(auditData)

	var ownerEmail string
	var user models.User
	owner, err := user.GetUserByID(base.Db.Postgresql, ownerId)
	if err == nil {
		ownerEmail = owner.Email
	}

	if err := audit_utility.CreateAuditLog(
		base.Db.Postgresql,
		ownerId,
		ownerEmail,
		"user",
		models.ActionOrganisationLeft,
		models.ResourceUser,
		userId,
		"",
		string(auditDataJSON),
		fmt.Sprintf("User %s removed user %s from organisation %s", ownerEmail, userId, orgId),
		audit_utility.GetClientIP(c),
		c.GetHeader("User-Agent"),
		true,
	); err != nil {
		base.Logger.Error("Failed to create audit log for user leaving organisation", err)
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
	invite_status := c.Query("invite_status")

	if invite_status != "" && invite_status != "invited" && invite_status != "accepted" && invite_status != "all" {
		base.Logger.Error("invalid invite status", nil)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid invite status", "invite_status must be either 'pending', 'invited' or 'all'", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

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

	invitations, paginationResponse, err := organisation.GetOrganisationInvites(c, base.Db.Postgresql, userId, orgId, invite_status)
	if err != nil {
		base.Logger.Error("failed to fetch organisation invites", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch organisation invites", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("organisation invites fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "success", invitations, paginationResponse)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) AddMemberToOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")

	var req models.OrgUserCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("failed to bind request", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to bind request", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(req)
	if err != nil {
		base.Logger.Error("validation error", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "validation error", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.RoleID); err != nil {
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

	if _, err := uuid.Parse(req.UserID); err != nil {
		base.Logger.Error("invalid user id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", "failed to decode user id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := organisation.AddMemberToOrganisation(ownerId, orgId, req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("failed to add member", err)
		rd := utility.BuildErrorResponse(code, "error", "failed to add member", err.Error(), nil)
		c.JSON(code, rd)
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

	if !utility.ValidateTimeRange(req.TimeRange) {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", "invalid time range supplied, time range must be in format HH:MM AM/PM - HH:MM AM/PM and start time must be before end time", nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if !ValidateNotifOption(string(req.NotifyAbout)) {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", "invalid notify_about suppplied, notify_about must be one of all_new_messages, mentions or nothing", nil)
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
		base.Logger.Error("failed to parse organisation id: %w", err)
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

func ValidateNotifOption(option string) bool {
	switch models.NotificationOption(option) {
	case models.AllMessages, models.DirectMentions, models.Nothing:
		return true
	default:
		return false
	}
}

func (base *Controller) ChangeMemberActiveStatus(c *gin.Context) {
	var (
		req     models.ChangeMemberActiveStatus
		user_id = c.Param("user_id")
		org_id  = c.Param("org_id")
	)

	if _, err := uuid.Parse(user_id); err != nil {
		base.Logger.Error("failed to parse user id: %w", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", errors.New("failed to parse user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("failed to parse organisation id: %w", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", errors.New("failed to parse organisation id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Error("failed to parse request body: %w", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("validation failed: %w", err)
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	adminUserID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Error("failed to fetch logged in user ID(org admin): %w", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to get user ID", errors.New("failed to get user ID"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":        org_id,
		"user_id":       user_id,
		"admin_user_id": adminUserID.(string),
	}

	code, err := organisation.ChangeMemberActiveStatus(base.Db.Postgresql, c, req, ids)
	if err != nil {
		base.Logger.Error("failed to change member active status: %w", err)
		rd := utility.BuildErrorResponse(code, "error", "failed to change member active status", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	status := "deactivated"
	if req.Activate {
		status = "activated"
	}

	base.Logger.Info("user %s from organisation %s successfully", status, org_id)
	rd := utility.BuildSuccessResponse(code, "success", fmt.Sprintf("user %s successfully", status))
	c.JSON(code, rd)
}

func (base *Controller) SearchUsersInOrganisation(c *gin.Context) {
	orgId := c.Param("org_id")
	query := c.Query("query")

	if _, err := uuid.Parse(orgId); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if query == "" {
		base.Logger.Error("search query cannot be empty")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "search query cannot be empty", "failed to search users in organisation", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	users, err := organisation.SearchUsersInOrganisation(base.Db.Postgresql, orgId, query)
	if err != nil {
		base.Logger.Error("failed to search users in organisation", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to search users in organisation", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("users searched successfully in organisation")
	rd := utility.BuildSuccessResponse(http.StatusOK, "success", users)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateMemberRole(c *gin.Context) {
	orgId := c.Param("org_id")
	userId := c.Param("user_id")

	var req models.UpdateMemberRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("failed to bind request", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to bind request", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

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

	err := base.Validator.Struct(req)
	if err != nil {
		base.Logger.Error("validation error", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "validation error", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Error("failed to get user claims", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to get user claims", "failed to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		OrganisationID: orgId,
		UserID:         userId,
		OwnerID:        claims.(string),
		RoleID:         req.RoleID,
	}

	code, err := organisation.UpdateMemberRole(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("failed to update role", err)
		rd := utility.BuildErrorResponse(code, "error", "failed to update role", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("member role updated successfully")
	rd := utility.BuildSuccessResponse(code, "success", nil)
	c.JSON(code, rd)
}
