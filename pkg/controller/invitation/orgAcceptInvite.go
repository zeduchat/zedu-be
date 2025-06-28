package invitation

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GeneralInvitationVerify(c *gin.Context) {
	var (
		req models.VerifyShareableInvitationLink
	)

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
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to get user claims", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to get user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	response, code, err := invitation.GeneralInvitationVerify(base.Db, req, base.Logger, userId)
	if err != nil {
		base.Logger.Info("Failed to verify invitation", err.Error())
		if code == http.StatusConflict {
			rd := utility.BuildSuccessResponse(http.StatusOK, "User is already a member", map[string]any{
				"isMember": true,
				"message":  err.Error(),
			})
			c.JSON(http.StatusOK, rd)
			return
		}

		rd := utility.BuildErrorResponse(code, "error", fmt.Errorf("failed to verify invitation: %w", err).Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user invited successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User invited successfully", map[string]any{
		"isMember": false,
		"message":  response,
	})
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) OrganisationVerifyInvite(c *gin.Context) {
	var (
		req = models.VerifyInvitationLinkRequest{}
	)

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

	respData, code, err := invitation.VerifyInvitation(req, base.Db, c, base.Logger)
	if err != nil {
		base.Logger.Info("Failed to verify invitation", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user invited successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User invited successfully", respData)
	c.JSON(http.StatusOK, rd)

}
