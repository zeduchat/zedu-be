package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) OrganisationCreateInvite(c *gin.Context) {
	var (
		inviteReq models.InvitationCreateReq
	)

	if err := c.ShouldBindJSON(&inviteReq); err != nil {
		base.Logger.Info("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("unable to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	err := base.Validator.Struct(&inviteReq)
	if err != nil {
		base.Logger.Info("Request Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Request Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	statusCode, msg, err := invitation.CheckerValidator(base.Db, inviteReq.Emails, inviteReq.OrganisationID, userId, base.Logger)
	if err != nil {
		base.Logger.Info("Failed to validate user", err)
		rd := utility.BuildErrorResponse(statusCode, "error", msg, err, nil)
		c.JSON(statusCode, rd)
		return
	}

	url := c.Request.Header.Get("Referer")

	inviteMap, err := invitation.InvitationLinkGenerator(base.Db, inviteReq, userId, url)
	if err != nil {
		base.Logger.Info("Failed to generate invitation link mapping", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to generate invitation link mapping", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	if _, err := uuid.Parse(inviteReq.Role); err != nil {
		base.Logger.Error("invalid role id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid role id format", "failed to decode role id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	// save invitations
	err = invitation.SaveInvitations(base.Db.Postgresql, inviteMap)
	if err != nil {
		base.Logger.Info("Failed to save invitations", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to save invitations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	mapData := invitation.InviteLinkMapper(url, inviteMap)

	//integrating send invitation functionality
	err = invitation.SendInvitationsEmail(base.Logger, mapData)
	if err != nil {
		base.Logger.Info("Failed to send invitation email", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to send invitation email", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusCreated, "Invitations created successfully", mapData)
	c.JSON(http.StatusCreated, rd)
}
