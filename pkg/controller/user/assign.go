package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	service "github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AssignRoleToUser(c *gin.Context) {
	userID := c.Param("user_id")
	roleID := c.Param("role_id")

	userData, err := service.ReplaceUserRole(userID, roleID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Role updated successfully", userData)
	c.JSON(http.StatusOK, rd)
}
