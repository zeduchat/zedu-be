package thread

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetAllUserOrgThreads(c *gin.Context) {

	var (
		userID = c.Param("user_id")
		orgID  = c.Param("org_id")
	)

	usersData, paginationResponse, code, err := service.GetAllUserOrgThreads(userID, orgID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAllUserChannelThreads(c *gin.Context) {

	var (
		userID    = c.Param("user_id")
		channelID = c.Param("channel_id")
	)

	usersData, paginationResponse, code, err := service.GetAllUserChannelThreads(userID, channelID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetUserSingleThreads(c *gin.Context) {

	var (
		userID   = c.Param("user_id")
		threadID = c.Param("thread_id")
	)

	usersData, code, err := service.GetUserSingleThreads(userID, threadID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) UpdateAThread(c *gin.Context) {

	var (
		userID   = c.Param("user_id")
		threadID = c.Param("thread_id")
		req      = models.UpdateThreadStatus{}
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

	code, err := service.UpdateAThread(req, userID, threadID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Thread updated successfully", nil)
	c.JSON(http.StatusOK, rd)

}
