package thread

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

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
		orgID = c.Param("org_id")
	)

	if _, err := uuid.Parse(orgID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid org id format", errors.New("failed to parse org id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	usersData, paginationResponse, code, err := service.GetAllUserOrgThreads(orgID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAllChannelThreads(c *gin.Context) {

	var (
		channelID = c.Param("channel_id")
	)

	if _, err := uuid.Parse(channelID); err != nil {
		base.Logger.Error(fmt.Sprintf("invalid channel id format: %v", err))
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	usersData, paginationResponse, code, err := service.GetAllChannelThreads(channelID, base.Db.Postgresql, c, base.Logger)
	if err != nil {
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Data retrieved successfully for channel threads")
	rd := utility.BuildSuccessResponse(code, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(code, rd)

}

func (base *Controller) GetChannelThreads(c *gin.Context) {

	var (
		channelID = c.Param("channel_id")
	)

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	usersData, paginationResponse, code, err := service.GetChannelThreads(channelID, base.Db.Postgresql, c, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData, paginationResponse)

	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetUserSingleThreads(c *gin.Context) {

	var (
		threadID  = c.Param("thread_id")
		channelID = c.Param("channel_id")
	)

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(threadID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid thread id format", errors.New("failed to parse thread id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	usersData, paginationResponse, code, err := service.GetUserSingleThreads(threadID, channelID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData, paginationResponse)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) UpdateAThread(c *gin.Context) {

	var (
		threadID  = c.Param("thread_id")
		channelID = c.Param("channel_id")
		req       = models.UpdateThreadStatus{}
	)

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(threadID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid thread id format", errors.New("failed to parse thread id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

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

	code, err := service.UpdateAThread(req, threadID, channelID, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Thread updated successfully", nil)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) DeleteAThread(c *gin.Context) {

	var (
		threadID  = c.Param("thread_id")
		channelID = c.Param("channel_id")
	)

	if _, err := uuid.Parse(channelID); err != nil {
		base.Logger.Error(fmt.Sprintf("invalid channel id format: %v", err))
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(threadID); err != nil {
		base.Logger.Error(fmt.Sprintf("invalid thread id format: %v", err))
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid thread id format", errors.New("failed to parse thread id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := service.DeleteAThread(threadID, channelID, base.Db.Postgresql, c, base.Logger)
	if err != nil {
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	base.Logger.Info("Thread deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Thread deleted successfully", nil)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) UpdateThreadMessage(c *gin.Context) {

	var (
		threadID  = c.Param("thread_id")
		channelID = c.Param("channel_id")
		req       = models.UpdateThreadMessage{}
	)

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	req.ChannelId = channelID

	if _, err := uuid.Parse(threadID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid thread id format", errors.New("failed to parse thread id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.ThreadId = threadID

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

	resp, code, err := service.UpdateThreadMessage(req, base.Db.Postgresql, c, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Thread updated successfully", resp)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) AddAThread(c *gin.Context) {

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

	ThreadData, err := service.CreateThreadMessage(req, base.Db, base.Logger)
	if err != nil {
		base.Logger.Info("some error occurred while creating thread: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		base.Logger.Error(fmt.Sprintf("an error occurred while processing request: %v", err))
		return
	}

	base.Logger.Info("thread message added successfully")

	rd := utility.BuildSuccessResponse(http.StatusCreated, "Thread message added successfully", ThreadData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) SearchChannel(c *gin.Context) {

	var (
		channelID = c.Param("channel_id")
		searching = c.Param("searching")
	)

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	usersData, code, err := service.SearchChannel(channelID, searching, base.Db.Postgresql, c, base.Db.TypeSense)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), nil, nil)
		c.JSON(code, rd)
		return
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", usersData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetChannelCountInfo(c *gin.Context) {
	var (
		orgID = c.Param("org_id")
	)

	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid days parameter", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	isValid := utility.IsValidUUID(orgID)
	if !isValid {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "Invalid Organisation UUID", "The provided organisation ID is not a valid UUID", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	usersData, channelMetrics, err := service.ChannelCountInfo(c, base.Db, orgID, days)
	if err != nil {
		base.Logger.Error("an error occurred while getting channel count metrics: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	response := gin.H{
		"channel_count_info": usersData,
		"channel_metrics":    channelMetrics,
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "Data retrieved successfully", response)
	c.JSON(http.StatusOK, rd)

}
