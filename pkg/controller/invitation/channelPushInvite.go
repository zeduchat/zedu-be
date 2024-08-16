package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) ChannelCreateInvite(c *gin.Context) {
	var (
		req = models.ChannelInvitationCreateReq{}
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

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("unable to get user claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	statusCode, msg, err := invitation.ChannelCheckerValidator(base.Db, req, userId, base.Logger)
	if err != nil {
		base.Logger.Info("Failed to validate user", err)
		rd := utility.BuildErrorResponse(statusCode, "error", msg, err, nil)
		c.JSON(statusCode, rd)
		return
	}

	url := c.Request.Header.Get("Base-Url")

	inviteMap, err := invitation.ChannelInvitationLinkGenerator(base.Db, req, userId, url)
	if err != nil {
		base.Logger.Info("Failed to generate channel invitation link mapping", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to generate channel invitation link mapping", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	// save invitations
	err = invitation.SaveChannelInvitations(base.Db.Postgresql, inviteMap)
	if err != nil {
		base.Logger.Info("Failed to save invitations", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to save invitations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	mapData := invitation.ChannelInviteLinkMapper(url, inviteMap)

	// integrating send invitation functionality
	err = invitation.SendChannelsInvitationsEmail(mapData)
	if err != nil {
		base.Logger.Info("Failed to send invitation email", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to send invitation email", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("channel invitation link sent to email")

	rd := utility.BuildSuccessResponse(http.StatusCreated, "Invitations created successfully", mapData)
	c.JSON(http.StatusCreated, rd)

}
