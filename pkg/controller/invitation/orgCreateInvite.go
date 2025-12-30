package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GeneralInvitationCreate(c *gin.Context) {
	var req models.ShareableInviteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Request Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to generate invitation link", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			base.Logger.Error("error validating user_id", err)
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "user claims not found", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		base.Logger.Error("error validating user_id", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "error validating user_id", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	//baseurl
	url := c.Request.Header.Get("Referer")

	response, code, err := invitation.GeneralInvitationCreate(base.Db.Postgresql, req, userID.(string), url)
	if err != nil {
		base.Logger.Error("Failed to create invitation link", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to create invitation link", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusCreated, "Invitation created successfully", response)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) OrganisationInviteMany(c *gin.Context) {
	var (
		inviteReq models.InvitationCreateReq
	)

	if err := c.ShouldBindJSON(&inviteReq); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("unable to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	// if !plan.CheckUserOrgPlanThreshold(c, base.Logger, base.Db.Postgresql, inviteReq.OrganisationID) {
	// 	base.Logger.Error("Maximum number of users for org plan reached!!")
	// 	rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "You have reached the maximum number of users for your organization plan", "Plan Limit Reached", nil)
	// 	c.JSON(http.StatusForbidden, rd)
	// 	return
	// }

	err := base.Validator.Struct(&inviteReq)
	if err != nil {
		base.Logger.Error("Request Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Request Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		OrganisationID: inviteReq.OrganisationID,
		UserID:         userId,
		RoleID:         inviteReq.RoleID,
	}

	statusCode, msg, err := invitation.CheckerValidator(base.Db, inviteReq.Emails, &ids, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to validate user", err)
		rd := utility.BuildErrorResponse(statusCode, "error", msg, err, nil)
		c.JSON(statusCode, rd)
		return
	}

	url := c.Request.Header.Get("Referer")

	inviteMap, errs, err := invitation.InvitationLinkGenerator(base.Db, inviteReq, userId, url)
	if err != nil {
		base.Logger.Error("Failed to generate invitation link mapping", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to generate invitation link mapping", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	if len(inviteMap) == 0 {
		base.Logger.Error("No invitations created. User(s) list is either empty or all invites have pending invitations")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "No invitations created. User(s) list is either empty or all invitees have pending invitations", errs, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = invitation.SaveInvitations(base.Db.Postgresql, inviteMap)
	if err != nil {
		base.Logger.Error("Failed to save invitations", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to save invitations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	mapData := invitation.InviteLinkMapper(url, inviteMap)

	err = invitation.SendInvitationsEmail(base.Db.Postgresql, base.Logger, mapData, url)
	if err != nil {
		base.Logger.Error("Failed to send invitation email", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to send invitation email", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	response := gin.H{
		"errors": errs,
	}

	
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Invitations created successfully", response)
	c.JSON(http.StatusCreated, rd)
}



func (base *Controller) OrganisationInviteFew(c *gin.Context) {
	var (
		inviteReq models.InvitationCreateFewRequest
	)

	if err := c.ShouldBindJSON(&inviteReq); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("unable to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	// if !plan.CheckUserOrgPlanThreshold(c, base.Logger, base.Db.Postgresql, inviteReq.OrganisationID) {
	// 	base.Logger.Error("Maximum number of users for org plan reached!!")
	// 	rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "You have reached the maximum number of users for your organization plan", "Plan Limit Reached", nil)
	// 	c.JSON(http.StatusForbidden, rd)
	// 	return
	// }

	err := base.Validator.Struct(&inviteReq)
	if err != nil {
		base.Logger.Error("Request Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Request Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		OrganisationID: inviteReq.OrganisationID,
		UserID:         userId,
	}

	statusCode, msg, err := invitation.FewInvitesCheckerValidator(base.Db, ids, inviteReq.Invitations)
	if err != nil {
		base.Logger.Error("Failed to validate user", err)
		rd := utility.BuildErrorResponse(statusCode, "error", msg, err.Error(), nil)
		c.JSON(statusCode, rd)
		return
	}

	url := c.Request.Header.Get("Referer")

	inviteMap, errs, err := invitation.InviteFewLinkGenerator(base.Db, inviteReq, userId, url)
	if err != nil {
		base.Logger.Error("Failed to generate invitation link mapping", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to generate invitation link mapping", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	if len(inviteMap) == 0 {
		base.Logger.Error("No invitations created. User(s) list is either empty or all invites have pending invitations")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "No invitations created. User(s) list is either empty or all invitees have pending invitations", errs, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = invitation.SaveInvitations(base.Db.Postgresql, inviteMap)
	if err != nil {
		base.Logger.Error("Failed to save invitations", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to save invitations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	mapData := invitation.InviteLinkMapper(url, inviteMap)

	err = invitation.SendInvitationsEmail(base.Db.Postgresql, base.Logger, mapData, url)
	if err != nil {
		base.Logger.Error("Failed to send invitation email", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to send invitation email", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	response := gin.H{
		"errors": errs,
	}

	
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Invitations created successfully", response)
	c.JSON(http.StatusCreated, rd)
}
