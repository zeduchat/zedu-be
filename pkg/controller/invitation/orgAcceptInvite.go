package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) OrganisationAcceptInvite(c *gin.Context) {
	// get accept invite logic here
	invitationToken := c.Param("t")
	claims, exists := c.Get("userClaims")
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	_, err := uuid.Parse(invitationToken)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid invitation token", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	invite, msg, err := invitation.AcceptInvitationLink(userId, invitationToken, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(
			http.StatusBadRequest,
			"error",
			msg,
			err,
			nil,
		)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	// add user to organisation
	///check if user from the claims is a member of the organisation
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	err = invitation.AddUserToOrganisation(base.Db.Postgresql, invite.OrganisationID, userId)
	if err != nil {
		base.Logger.Error("Failed to add user to organisation", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "A server error occurred", nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	rd := utility.BuildSuccessResponse(http.StatusOK, "Invitation accepted successfully", nil)
	c.JSON(http.StatusOK, rd)
}
