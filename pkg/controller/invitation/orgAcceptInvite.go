package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

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

	respData, code, err := invitation.VerifyInvitation(req, base.Db.Postgresql, c)
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
