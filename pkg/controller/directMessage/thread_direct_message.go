package dm

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	dm "github.com/hngprojects/telex_be/services/directMessage"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AddAThreadDm(c *gin.Context) {

	var (
		req       = models.CreateThreadMsgReq{}
		channelID = c.Param("channel_id")
	)

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
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

	req.ChannelsID = channelID

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	req.UserId = userClaims["user_id"].(string)

	ThreadData, code, err := dm.CreateThreadDmMessage(req, base.Db, base.Logger)
	if err != nil {
		base.Logger.Info("some error occurred while creating thread: " + err.Error())
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	base.Logger.Info("thread message added successfully")

	rd := utility.BuildSuccessResponse(code, "Thread message added successfully", ThreadData)
	c.JSON(code, rd)

}

func (base *Controller) GetAllChannelThreads(c *gin.Context) {

	var (
		channelID = c.Param("channel_id")
	)

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	usersData, paginationResponse, code, err := dm.GetAllChannelDmThreads(channelID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(code, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(code, rd)

}
