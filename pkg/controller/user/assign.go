package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AssignRoleToUser(c *gin.Context) {
	var (
		req = models.SwitchUserRoleRequest{}
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

	userData, code, err := service.ReplaceUserRole(req.UserId, req.OrgId, req.RoleId, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
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
