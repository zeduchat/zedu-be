package huddle

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) SetActiveScreen(c *gin.Context) {
	huddleID := c.Param("id")
	if huddleID == "" {
		base.Logger.Info("missing huddle ID")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "huddle ID is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.HuddleSetActiveScreen

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing huddle note request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
}
