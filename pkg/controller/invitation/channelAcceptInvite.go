package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) ChannelVerifyInvite(c *gin.Context) {
	var (
		req = models.VerifyInvitationLinkRequest{}
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

	respData, code, err := invitation.VerifyChannelInvitation(req, base.Db.Postgresql, c, base.ExtReq)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user invited successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User invited successfully", respData)
	c.JSON(http.StatusOK, rd)
}
