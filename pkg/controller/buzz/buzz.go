package buzz

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

func (base *Controller) Create(c *gin.Context) {
	var req models.CreateBuzzRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing buzz request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for buzz request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzz.CreateBuzz(base.Db, base.Logger, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to create buzz: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("buzz created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "buzz created successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) Join(c *gin.Context) {
	buzzCodeOrID := c.Param("id")

	if !utility.IsValidBuzzCodeOrUUID(buzzCodeOrID) {
		base.Logger.Info("invalid buzz code or ID format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code or ID format", "failed to decode buzz code or ID", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCodeOrID)
	if err != nil {
		base.Logger.Info("buzz not found for code: %s", buzzCodeOrID)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	resp, code, err := buzz.JoinBuzz(base.Db, base.Logger, buzzID, userID.(string))
	if err != nil {
		base.Logger.Error("failed to join buzz: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user joined buzz successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "user joined buzz successfully", resp)
	c.JSON(http.StatusOK, rd)
}

// GetAgoraToken generates an Agora RTC token for joining a buzz
func (base *Controller) GetAgoraToken(c *gin.Context) {
	var req models.BuzzAgoraTokenRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing Agora token request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for Agora token request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := agora.GetAgoraToken(base.Db, base.Logger, req.BuzzID, userID.(string), req.UID)
	if err != nil {
		base.Logger.Error("failed to generate Agora token: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Agora token generated successfully for UID: %s", req.UID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agora token generated successfully", resp)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) LeaveBuzz(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		base.Logger.Error("invalid request param: buzz code is invalid")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code in params", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		base.Logger.Error("buzz not found for code: %s", buzzCode)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	data, statusCode, err := buzz.LeaveBuzz(base.Db, base.Logger, buzzID, userID)

	if err != nil {
		base.Logger.Error("Failed to leave buzz: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("user %s left buzz %s successfully", userID, buzzID)
	rd := utility.BuildSuccessResponse(statusCode, "user left buzz successfully", data)
	c.JSON(statusCode, rd)
}

func (base *Controller) EndBuzz(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		base.Logger.Error("invalid request param: buzz code is invalid")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code in params", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		base.Logger.Error("buzz not found for code: %s", buzzCode)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	data, statusCode, err := buzz.EndBuzz(base.Db, base.Logger, buzzID, userID)

	if err != nil {
		base.Logger.Error("Failed to end buzz: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("buzz %s ended by host %s successfully", buzzID, userID)
	rd := utility.BuildSuccessResponse(statusCode, "buzz ended successfully", data)
	c.JSON(statusCode, rd)
}

func (base *Controller) EndBuzzByChannel(c *gin.Context) {
	channelID, ok := c.Params.Get("channel_id")
	if !ok || !(utility.IsValidUUID(channelID)) {
		base.Logger.Error("invalid request param: channel id is invalid")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id in params", errors.New("invalid channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	data, statusCode, err := buzz.EndBuzzByChannel(base.Db, base.Logger, channelID, userID)

	if err != nil {
		base.Logger.Error("Failed to end buzz by channel: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("buzz in channel %s ended by host %s successfully", channelID, userID)
	rd := utility.BuildSuccessResponse(statusCode, "buzz ended successfully", data)
	c.JSON(statusCode, rd)
}

func (base *Controller) GetMetadata(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		base.Logger.Error("invalid request param: buzz code is invalid")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code in params", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		base.Logger.Error("buzz not found for code: %s", buzzCode)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	data, statusCode, err := buzz.GetBuzzMetadata(base.Db, base.Logger, buzzID, userID)

	if err != nil {
		base.Logger.Error("Failed to fetch buzz metadata: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("buzz metadata retrieved successfully for buzz %s", buzzID)
	rd := utility.BuildSuccessResponse(statusCode, "buzz metadata retrieved successfully", data)
	c.JSON(statusCode, rd)
}

func (base *Controller) GetChannelActiveBuzz(c *gin.Context) {
	base.getActiveBuzzForChannel(c, "channel_id", "channel")
}

func (base *Controller) GetDMActiveBuzz(c *gin.Context) {
	base.getActiveBuzzForChannel(c, "dm_id", "dm")
}

func (base *Controller) GetGroupDMActiveBuzz(c *gin.Context) {
	base.getActiveBuzzForChannel(c, "group_dm_id", "group dm")
}

func (base *Controller) getActiveBuzzForChannel(c *gin.Context, paramName, channelType string) {
	channelID := c.Param(paramName)

	if channelID == "" || !(utility.IsValidUUID(channelID)) {
		base.Logger.Error("invalid request param: %s id is empty or invalid", channelType)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", fmt.Sprintf("invalid %s id in params", channelType), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	data, statusCode, err := buzz.GetChannelActiveBuzzIndicator(base.Db, base.Logger, channelID, userID.(string))

	if err != nil {
		base.Logger.Error("Failed to fetch active buzz indicator: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("active buzz indicator retrieved for %s %s", channelType, channelID)
	rd := utility.BuildSuccessResponse(statusCode, "active buzz status retrieved", data)
	c.JSON(statusCode, rd)
}

// ForceEndBuzz force ends a buzz without permission checks - FOR TESTING ONLY
func (base *Controller) ForceEndBuzz(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		base.Logger.Error("invalid request param: buzz code is invalid")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code in params", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		base.Logger.Error("buzz not found for code: %s", buzzCode)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	base.Logger.Warning("[TEST ENDPOINT] Force ending buzz %s via test endpoint", buzzID)

	data, statusCode, err := buzz.ForceEndBuzz(base.Db, base.Logger, buzzID)

	if err != nil {
		base.Logger.Error("Failed to force end buzz: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("[TEST ENDPOINT] buzz %s force ended successfully", buzzID)
	rd := utility.BuildSuccessResponse(statusCode, "buzz force ended successfully (test endpoint)", data)
	c.JSON(statusCode, rd)
}

func (base *Controller) CreateOrgBuzz(c *gin.Context) {

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	orgID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "org_id")
	if err != nil {
		base.Logger.Info("unable to fetch org claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "organization context required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	resp, code, err := buzz.CreateOrgBuzz(base.Db, base.Logger, userID.(string), orgID.(string))
	if err != nil {
		base.Logger.Error("failed to create org buzz: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("org buzz created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "organization buzz created successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetOrgBuzzList(c *gin.Context) {
	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	orgID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "org_id")
	if err != nil {
		base.Logger.Info("unable to fetch org claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "organization context required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	resp, code, err := buzz.GetOrgBuzzList(base.Db, base.Logger, userID.(string), orgID.(string))
	if err != nil {
		base.Logger.Error("failed to fetch org buzz list: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("org buzz list retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "organization buzz list retrieved successfully", resp)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateMediaState(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		base.Logger.Error("invalid request param: buzz code is invalid")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code in params", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		base.Logger.Error("buzz not found for code: %s", buzzCode)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var req models.UpdateMediaStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing media state request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for media state request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := buzz.UpdateMediaState(base.Db, base.Logger, buzzID, userID, req.MediaState)
	if err != nil {
		base.Logger.Error("failed to update media state: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("media state updated successfully for user %s in buzz %s", userID, buzzID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "media state updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) SendBuzzMessage(c *gin.Context) {
	var (
		req      models.SendBuzzMessageRequest
		buzzCode = c.Param("id")
	)

	if !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		base.Logger.Error("invalid buzz code format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code format", errors.New("failed to parse buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		base.Logger.Error("buzz not found for code: %s", buzzCode)
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	err = c.ShouldBind(&req)
	if err != nil {
		base.Logger.Error("Failed to parse request body: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Validation failed: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed",
			utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Error("Unable to get user claims: user not authorized")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userID := userClaims["user_id"].(string)

	var buzzRecord models.Buzz
	if err := base.Db.Postgresql.Where("id = ?", buzzID).First(&buzzRecord).Error; err != nil {
		base.Logger.Error("Failed to fetch buzz: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	if buzzRecord.Status != models.BuzzStatusActive {
		base.Logger.Error("Buzz is not active")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "buzz is not active", errors.New("buzz has ended"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	isParticipant := false
	for _, participantID := range buzzRecord.ParticipantIDs {
		if participantID == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		base.Logger.Error("User is not a participant in this buzz")
		rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "user is not a participant in this buzz", errors.New("not a participant"), nil)
		c.JSON(http.StatusForbidden, rd)
		return
	}

	var profile models.Profile
	err = profile.GetProfileByUserId(base.Db.Postgresql, userID)
	if err != nil {
		base.Logger.Error("Failed to get user profile: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to get user profile", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	var user models.User
	user, err = user.GetUserByID(base.Db.Postgresql, userID)
	if err != nil {
		base.Logger.Error("Failed to get user: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to get user", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	orgID := ""
	if buzzRecord.OrgID != nil {
		orgID = *buzzRecord.OrgID
	}

	feed := models.FeedMessageRequest{
		ChannelID:        buzzRecord.ChannelID,
		UserName:         profile.UserName,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		AvatarURL:        profile.AvatarURL,
		DefaultAvatarUrl: avatar.GenerateDefaultAvatarURL(userID),
		Type:             "buzz_message",
		Content:          req.Content,
		ThreadId:         buzzID,
		Email:            user.Email,
		UserType:         "user",
		FullName:         profile.FullName,
		UserId:           userID,
		OrgId:            orgID,
		BuzzCode:         utility.ExtractBuzzCode(buzzID),
		Media:            req.Media,
		ChannelName:      "",
		ChannelType:      buzzRecord.ChannelType,
	}

	publishChannel := buzzRecord.ChannelID
	if buzzRecord.BuzzType == models.BuzzTypeOrganization {
		publishChannel = utility.ExtractBuzzCode(buzzID)
	}

	err = centrifuge.PublishChannel(base.Logger, publishChannel, feed)
	if err != nil {
		base.Logger.Error(fmt.Sprintf("Error publishing to buzz channel: %s, error: %v", publishChannel, err.Error()))
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to publish message", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("buzz message published successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "message sent successfully", feed)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) MuteParticipants(c *gin.Context) {
	buzzCode, ok := c.Params.Get("id")
	if !ok || !utility.IsValidBuzzCodeOrUUID(buzzCode) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz code", errors.New("invalid buzz code"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	buzzID, err := utility.ResolveBuzzCode(base.Db.Postgresql, buzzCode)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "buzz not found", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid user id", errors.New("invalid user id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	code, err := buzz.MuteParticipants(base.Db, base.Logger, buzzID, userID)
	if err != nil {
		base.Logger.Error("failed to mute participants: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("muted participants for buzz %s by host %s", buzzID, userID)
	rd := utility.BuildSuccessResponse(http.StatusOK, "muted participants successfully", nil)
	c.JSON(http.StatusOK, rd)
}
