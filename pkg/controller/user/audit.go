package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetUserLoginAudit(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	if _, err := uuid.Parse(userID); err != nil {
		base.Logger.Info("error parsing user id")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", errors.New("failed to parse user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	usersData, paginationResponse, code, err := service.GetAUserLoginActivity(userID, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("error getting user login activity", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user login activity retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetUserSessions(c *gin.Context) {

	var (
		userID = c.Param("user_id")
	)

	if _, err := uuid.Parse(userID); err != nil {
		base.Logger.Info("error parsing user id")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id format", errors.New("failed to parse user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	usersData, paginationResponse, code, err := service.GetAUserSessions(userID, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("error getting user sessions", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user sessions retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) RevokeUserAccessToken(c *gin.Context) {
	var (
		req = models.TerminateSessionRequest{}
	)

	err := c.ShouldBind(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := service.RevokeUserAccessToken(req, base.Db.Postgresql, c)
	if err != nil {
		base.Logger.Error("error terminating session", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("session terminated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "User session terminated successfully", nil)
	c.JSON(http.StatusOK, rd)
}
