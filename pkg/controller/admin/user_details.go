package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	admin "github.com/hngprojects/telex_be/services/admin"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetUserDetails(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, code, err := admin.GetUserDetails(base.Db.Postgresql, userID)
	if err != nil {
		base.Logger.Error("failed to get user details", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user details retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "user details retrieved successfully", response)
	c.JSON(http.StatusOK, rd)
}
