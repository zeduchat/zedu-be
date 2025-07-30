package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AdminResend(c *gin.Context) {
	var (
		req models.ResendCondition
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	url := c.Request.Header.Get("Referer")

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Request Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Request Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := invitation.AdminResend(base.Db.Postgresql, base.Logger, req, url)
	if err != nil {
		base.Logger.Error("Request Validation failed", err)
		rd := utility.BuildErrorResponse(code, "error", "Request Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Invitations created successfully", nil)
	c.JSON(code, rd)
}

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
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "unauthorized user", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	userId := userID.(string)

	url := c.Request.Header.Get("Referer")

	emails := []string{req.Email}

	ids := models.IDS{
		OrganisationID: req.OrganisationID,
		UserID:         userId,
	}

	statusCode, msg, err := invitation.CheckerValidator(base.Db, emails, ids, base.Logger)
	if err != nil {
		base.Logger.Error("Failed to validate user", err)
		rd := utility.BuildErrorResponse(statusCode, "error", msg, err.Error(), nil)
		c.JSON(statusCode, rd)
		return
	}

	response, err := invitation.ResendLinkGenerator(base.Db, base.Logger, req)
	if err != nil {
		base.Logger.Error("Failed to generate invitation resend link", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to generate invitation resend link", "", err)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	mapData := invitation.InviteLinkMapper(url, []models.Invitation{response})

	err = invitation.SendInvitationsEmail(base.Db.Postgresql, base.Logger, mapData, url)
	if err != nil {
		base.Logger.Error("Failed to send invitation", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to send invitation", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "success", "Invitation resent successfully", nil)
	c.JSON(http.StatusOK, rd)
}
