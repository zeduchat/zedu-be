package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) ResendInvitation(c *gin.Context) {
	var req models.ResendInvitationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Error binding request", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error ", "Invalid request", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Request validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Request validation failed", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to create blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to create blog", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	url := c.Request.Header.Get("Referer")

	statusCode, msg, err := invitation.CheckerValidator(base.Db, req.Emails, req.OrganisationID, userId, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to validate user", err)
		rd := utility.BuildErrorResponse(statusCode, "error", msg, err.Error(), nil)
		c.JSON(statusCode, rd)
		return
	}

	response, err := invitation.ResendLinkGenerator(base.Db, base.Logger, req, userId)
	if err != nil {
		base.Logger.Error("Failed to generate invitation link mapping", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to generate invitation link mapping", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	mapData := invitation.InviteLinkMapper(url, response)

	err = invitation.SendInvitationsEmail(base.Logger, mapData)
	if err != nil {
		base.Logger.Error("Failed to send invitations", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to send invitations", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", "Invitations sent successfully", mapData)
	c.JSON(http.StatusOK, rd)
}
