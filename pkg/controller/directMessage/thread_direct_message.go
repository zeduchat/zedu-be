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

	threadData, statusCode, err := dm.CreateThreadDmMessage(req, base.Db, base.Logger)
	if err != nil {
		base.Logger.Info("some error occurred while creating thread: " + err.Error())
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), nil, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("dm thread message added successfully")
	rd := utility.BuildSuccessResponse(statusCode, "dm thread message added successfully", threadData)
	c.JSON(statusCode, rd)
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

func (base *Controller) BotDMResponse(c *gin.Context) {
	var req models.BotReturnRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := dm.BotResponse(req, base.Db, base.Logger, base.ExtReq)
	if err != nil {
		base.Logger.Error("error creating dm channel", err)
		rd := utility.BuildErrorResponse(code, "error", "failed to post message", err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Bot response sent successfully")
	rd := utility.BuildSuccessResponse(code, "Bot response sent successfully", respData)
	c.JSON(code, rd)
}
