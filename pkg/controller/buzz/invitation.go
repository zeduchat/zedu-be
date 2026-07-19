package buzz

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
)

// SearchChannelMembers handles searching for channel members to invite
func (base *Controller) SearchChannelMembers(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	var req models.SearchChannelMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing search members request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for search members request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzz.SearchChannelMembers(base.Db, base.Logger, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to search channel members: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("channel members retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "members retrieved successfully", resp)
	c.JSON(http.StatusOK, rd)
}

// InviteUsersToBuzz handles inviting users to a buzz
func (base *Controller) InviteUsersToBuzz(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	var req models.InviteUsersToBuzzRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing invite users request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for invite users request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzz.InviteUsersToBuzz(base.Db, base.Logger, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to invite users to buzz: %v", err)
		rd := utility.BuildErrorResponse(code, "error", "failed to send any invitations", err.Error(), resp)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("users invited to buzz successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, resp.Message, resp)
	c.JSON(http.StatusCreated, rd)
}

// RespondToInvitation handles accepting or declining a buzz invitation
func (base *Controller) RespondToInvitation(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	var req models.RespondToInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing respond to invitation request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for respond to invitation request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzz.RespondToInvitation(base.Db, base.Logger, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to respond to invitation: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("invitation responded to successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, resp.Message, resp)
	c.JSON(http.StatusOK, rd)
}

// GetPendingInvitations retrieves all pending invitations for the current user
func (base *Controller) GetPendingInvitations(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	responses, code, err := buzz.GetPendingInvitations(base.Db, base.Logger, userID.(string))
	if err != nil {
		base.Logger.Error("failed to fetch pending invitations: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	responseData := map[string]interface{}{
		"invitations": responses,
		"total":       len(responses),
	}

	base.Logger.Info("pending invitations retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "pending invitations retrieved", responseData)
	c.JSON(http.StatusOK, rd)
}
