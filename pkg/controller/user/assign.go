package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	service "github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AssignRoleToUser(c *gin.Context) {
	userID := c.Param("user_id")
	roleID := c.Param("role_id")

	if _, err := uuid.Parse(userID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", errors.New("failed to parse user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(roleID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid role id format", errors.New("failed to parse role id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userData, err := service.ReplaceUserRole(userID, roleID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Role updated successfully", userData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateUserIdentity(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	roleID := ctx.Param("role_id")

	userData, code, err := service.UpdateUserIdentity(userID, roleID, base.Db.Postgresql, ctx)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), nil, nil)
		ctx.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Role updated successfully", userData)
	ctx.JSON(code, rd)
}
