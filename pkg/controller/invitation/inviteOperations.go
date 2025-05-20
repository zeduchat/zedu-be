package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) ChangeGeneralInviteStatus(c *gin.Context) {
	var req models.ChangeStatus

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("Validation failed", err)
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
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

	code, err := invitation.ChangeGeneralInviteStatus(base.Db.Postgresql, req, base.Logger, userId)
	if err != nil {
		base.Logger.Error("Failed to change invitation status")
		rd := utility.BuildErrorResponse(code, "error", "Failed to change invitation status", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Invitation updated successfully")
	rd := utility.BuildSuccessResponse(code, "invitation updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) CancelInvitation(c *gin.Context) {
	inviteID := c.Param("invite_id")

	if _, err := uuid.Parse(inviteID); err != nil {
		base.Logger.Error("invalid invitation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid invitation id format", "failed to delete invitation", nil)
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

	userId := userID.(string)

	err = invitation.CancelInvitation(base.Db.Postgresql, inviteID, userId)

	if err != nil {
		base.Logger.Error("failed to cancel invitation", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to cancel invitation", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "invitation cancelled successfully", nil)
	c.JSON(http.StatusOK, rd)
}
