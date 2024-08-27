package invitation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GenerateGlobalInvitation(c *gin.Context) {
	var inviteReq models.GlobalInviteRequest

	if err := c.ShouldBindJSON(&inviteReq); err != nil {
		base.Logger.Info("Failed to parse request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	
}
