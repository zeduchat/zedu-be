package channel

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/channel"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateChannel(c *gin.Context) {
	var req models.CreateChannelsRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.OrganisationID); err != nil {
		base.Logger.Info("error parsing organisation id")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid organisation id format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	req.UserId = userId

	// if !plan.CheckChannelPlanThreshold(c, base.Logger, base.Db.Postgresql, req.OrganisationID) {
	// 	base.Logger.Error("Maximum number of channels for org plan reached!!")
	// 	rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "You have reached the maximum number of channels for your organization plan", "Plan Limit Reached", nil)
	// 	c.JSON(http.StatusForbidden, rd)
	// 	return
	// }

	respData, code, err := channel.CreateChannel(req, base.Db, base.Logger)
	if err != nil {
		base.Logger.Error("error creating channel", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("channel created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Channel Created Successfully", respData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetChannel(c *gin.Context) {
	channels_id := c.Param("channelId")

	if _, err := uuid.Parse(channels_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := models.IDS{
		UserID:    userId,
		ChannelID: channels_id,
	}

	respData, code, err := channel.GetChannel(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Info("error getting channel")
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("channel retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel retreived successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetChannelsMsg(c *gin.Context) {

	ChannelsId := c.Param("channelId")

	if _, err := uuid.Parse(ChannelsId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	UserId := userClaims["user_id"].(string)

	respData, code, err := channel.GetChannelsMsg(ChannelsId, UserId, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("channel messages fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel messages fetched successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) JoinChannels(c *gin.Context) {
	var (
		req models.JoinChannelsRequest
	)

	channels_id := c.Param("channelId")

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	user_id := userClaims["user_id"].(string)

	if req.ChannelsID == "" {
		req.ChannelsID = channels_id
	}
	if req.UserID == "" {
		req.UserID = user_id
	}

	newReq := models.JoinChannelsRequest{
		Username:   req.Username,
		ChannelsID: req.ChannelsID,
		UserID:     req.UserID,
	}

	err := c.ShouldBindJSON(&newReq)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&newReq)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	channel, code, err := channel.JoinChannels(base.Db, newReq, base.Logger)
	if err != nil {
		base.Logger.Info("error joining channel")
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("channel joined successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel joined successfully", channel)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) LeaveChannels(c *gin.Context) {

	channelId := c.Param("channelId")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	user_id := userClaims["user_id"].(string)

	code, err := channel.LeaveChannels(base.Db, channelId, user_id, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user left channel successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "user left channel successfully", gin.H{})
	c.JSON(code, rd)
}

func (base *Controller) UpdateUsername(c *gin.Context) {
	var req models.UpdateChannelsUserNameReq

	channelId := c.Param("channelId")

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if req.Username == "general" {
		base.Logger.Info("error updating username")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "cannot update username to general", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := channel.UpdateUsername(req, base.Db.Postgresql, channelId, userId)
	if err != nil {
		base.Logger.Info("error creating channel")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("username updated successfully")
	rd := utility.BuildSuccessResponse(code, "username updated successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) DeleteChannel(c *gin.Context) {

	ChannelsId := c.Param("channelId")
	if _, err := uuid.Parse(ChannelsId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	UserId := userClaims["user_id"].(string)

	code, err := channel.DeleteChannel(base.Db.Postgresql, ChannelsId, UserId)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("channel deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel deleted successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) GetChannelsByName(c *gin.Context) {
	name := c.Params.ByName("channelName")

	respData, code, err := channel.GetChannelsByName(base.Db.Postgresql, name)
	if err != nil {
		base.Logger.Info("error getting channel")
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("channel name retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel name retrieved successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CountChannelsUsers(c *gin.Context) {
	channelId := c.Param("channelId")

	if _, err := uuid.Parse(channelId); err != nil {
		base.Logger.Info("failed to get channelId")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	totalCount, code, err := channel.CountChannelsUsers(base.Db.Postgresql, channelId)
	if err != nil {
		base.Logger.Info("error getting total channel users")
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("channel users count retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel users count retrieved successfully", totalCount)
	c.JSON(code, rd)
}

func (base *Controller) UpdateChannels(c *gin.Context) {
	id := c.Param("channelId")
	var req models.UpdateChannelsRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims: user not authorized")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	if _, err := uuid.Parse(id); err != nil {
		base.Logger.Info("error parsing channel id: %v", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid ID format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error binding request body: %v", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("error validating request: %v", err)
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	if req.Name == "general" {
		base.Logger.Info("error: attempt to update channel name to general")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Cannot update channel name to general", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := models.IDS{
		ChannelID: id,
		UserID:    userId,
	}

	result, code, err := channel.UpdateChannels(base.Db.Postgresql, req, ids)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			base.Logger.Info("error: channel not found: %v", err.Error())
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", fmt.Sprintf("Channels not found: %v", err.Error()), err.Error(), nil)
			c.JSON(code, rd)
		} else {
			base.Logger.Info("error updating channel: %v", err.Error())
			rd := utility.BuildErrorResponse(code, "error", fmt.Sprintf("Failed to update channel: %v", err.Error()), err.Error(), nil)
			c.JSON(code, rd)
		}
		return
	}

	base.Logger.Info("Channels updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channels updated successfully", result)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CheckUser(c *gin.Context) {

	ChannelsId := c.Param("channelId")

	if _, err := uuid.Parse(ChannelsId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	UserId := userClaims["user_id"].(string)

	respData, code, err := channel.CheckUser(ChannelsId, UserId, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user checked successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "user checked successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) SearchChannelsByNames(c *gin.Context) {
	name := c.Param("channelName")

	channels, paginationResponse, err := channel.SearchChannelsByNames(base.Db.Postgresql, c, name)
	if err != nil {
		base.Logger.Info("error fetching channels")
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "failed to fetch channels", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("channel names retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel names retrieved successfully", channels, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetUsersInChannel(c *gin.Context) {
	channelID := c.Param("channelId")

	if _, err := uuid.Parse(channelID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channels id format", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	users, paginationResponse, err := channel.GetUsersInChannel(channelID, userId, base.Db.Postgresql, c)

	if err != nil {
		switch err.Error() {
		case "channels not found":
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to retrieve users", nil)
			c.JSON(http.StatusNotFound, rd)
		case "user does not have access to the channels":
			rd := utility.BuildErrorResponse(http.StatusForbidden, "error", err.Error(), "failed to retrieve users", nil)
			c.JSON(http.StatusNotFound, rd)
		default:
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to retrieve users", err.Error(), nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("users retrieved successfully")
	response := utility.BuildSuccessResponse(http.StatusOK, "users retrieved successfully", users, paginationData)

	c.JSON(http.StatusOK, response)
}

func (base *Controller) AddMembersToChannel(c *gin.Context) {
	var req models.JoinChannelsRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(req.ChannelsID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, err := channel.AddMembersToChannel(base.Db, req, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("members added to channel successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Members added to channel successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetUserChannels(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to fetch user claims", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to fetch user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	ids := models.IDS{
		OrganisationID: org_id,
		UserID:         userId,
	}

	userchannels, err := channel.GetUserChannels(base.Db, ids)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch user channels", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("user channels fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "user channels fetched successfully", userchannels)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetUserNotInChannels(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to fetch user claims", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to fetch user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	ids := models.IDS{
		OrganisationID: org_id,
		UserID:         userId,
	}

	userchannels, err := channel.GetUserNotInChannels(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Info("failed to fetch channels user do not belong")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch channels user do not belong", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("channels user does not belong fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channels user does not belong fetched successfully", userchannels)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) AddMultipleMembersToChannel(c *gin.Context) {
	var req models.AddMultipleMembersRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to fetch user claims", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to fetch user claims", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	req.UserID = userID.(string)

	err = channel.AddMultipleMembersToChannel(base.Db, req, base.Logger)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to add users to channel", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("users added to channel successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "users added to channel successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ArchiveChannel(c *gin.Context) {
	channelId := c.Param("channelId")
	var req models.ArchiveChannelRequest

	if _, err := uuid.Parse(channelId); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to retrieve users", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = utility.ArchiveValidator(req.Archived)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", err.Error(), err, nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	status, statusCode, err := channel.ArchiveChannel(base.Db.Postgresql, channelId, req)
	if err != nil {
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	res := "archived"
	if !status {
		res = "unarchived"
	}

	base.Logger.Info("channel archived successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "channel "+res+" successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetArchivedChannels(c *gin.Context) {
	var org_id string = c.Param("org_id")

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	ids := map[string]string{
		"organisation_id": org_id,
		"user_id":         userId,
	}

	respData, code, err := channel.GetArchivedChannels(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Info("error getting archived channels")
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("archived channels retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "archived channels retrieved successfully", respData)
	c.JSON(http.StatusOK, rd)
}
