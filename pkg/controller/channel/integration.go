package channel

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/channel"
	"github.com/hngprojects/telex_be/utility"
)

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

	req.ChannelID = channels_id

	err := c.ShouldBindJSON(&req)
	if err != nil {
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

	req.ChannelID = channels_id

	err := c.ShouldBindJSON(&req)
	if err != nil {
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
