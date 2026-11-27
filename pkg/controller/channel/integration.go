package channel

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/services/channel"
	"github.com/hngprojects/telex_be/services/organisation"
	"github.com/hngprojects/telex_be/utility"
)

// authorizeIntegrationChannel writes the error response and returns false unless
// the caller can access channelID and, when manage is set, holds
// can_manage_integrations (or owns) the channel's organisation. Non-members get
// 404 so the response does not reveal that the channel exists.
func (base *Controller) authorizeIntegrationChannel(c *gin.Context, channelID string, manage bool) bool {
	db := base.Db.Postgresql
	raw, err := middleware.GetUserClaims(c, db, "user_id")
	userID, ok := raw.(string)
	if err != nil || !ok || userID == "" {
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "unauthorized", "failed to fetch user claims", nil)
		c.JSON(http.StatusUnauthorized, rd)
		return false
	}

	allowed, err := models.UserCanAccessChannel(db, channelID, userID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to check channel access", nil, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return false
	}
	if !allowed {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "channel not found", nil, nil)
		c.JSON(http.StatusNotFound, rd)
		return false
	}

	if manage {
		var ch models.Channels
		err := db.Select("id", "organisation_id").Where("id = ?", channelID).Take(&ch).Error
		if err != nil || ch.OrganisationID == "" ||
			!organisation.UserCanOrOwner(db, userID, ch.OrganisationID, models.PermManageIntegrations) {
			rd := utility.BuildErrorResponse(http.StatusForbidden, "error", "user does not have permission to manage integrations in this channel", nil, nil)
			c.JSON(http.StatusForbidden, rd)
			return false
		}
	}
	return true
}

func (base *Controller) GetIntegrationChannels(c *gin.Context) {
	channels_id := c.Param("channelId")
	modifier_id := c.Param("IntModId")

	if _, err := uuid.Parse(channels_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(modifier_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid modifier id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if !base.authorizeIntegrationChannel(c, channels_id, false) {
		return
	}

	respData, code, err := channel.GetChannelIntegration(base.Db.Postgresql, channels_id, modifier_id)

	if err != nil {

		base.Logger.Info("error getting integration channels, err: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("integration channels retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "channel retreived successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) AddIntegrationChannel(c *gin.Context) {
	var (
		req models.AddIntegrationChannel
	)

	channels_id := c.Param("channelId")

	if _, err := uuid.Parse(channels_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if !base.authorizeIntegrationChannel(c, channels_id, true) {
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	// The path decides the target channel; ignore any channel_id in the body.
	req.ChannelID = channels_id

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	res, code, err := channel.AddIntegrationChannel(base.Db.Postgresql, req)
	if err != nil {
		base.Logger.Info("erro adding integration channel: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("integration channel added successfully")
	rd := utility.BuildSuccessResponse(code, "integration channel added successfully", res)
	c.JSON(code, rd)
}

func (base *Controller) DeleteChannelIntegration(c *gin.Context) {
	var (
		req models.IntegrationChannelReq
	)

	channels_id := c.Param("channelId")

	if _, err := uuid.Parse(channels_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", errors.New("failed to parse channel id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if !base.authorizeIntegrationChannel(c, channels_id, true) {
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	// The path decides the target channel; ignore any channel_id in the body.
	req.ChannelID = channels_id

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := channel.DeleteChannelIntegration(base.Db.Postgresql, req)
	if err != nil {
		base.Logger.Info("erro deleting integration channel: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("integration channel deleted successfully")
	rd := utility.BuildSuccessResponse(code, "integration channel deleted successfully", nil)
	c.JSON(code, rd)
}
