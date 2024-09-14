package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) GetOrganisationChannelIntegrations(c *gin.Context) {
	channel_id := c.Param("channel_id")
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to decode channel id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	integrations, paginationResponse, err := integrations.GetOrganisationChannelIntegrations(base.Db.Postgresql, channel_id, org_id, c)

	if err != nil {
		base.Logger.Error("Failed to get channel integrations")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to get channel integrations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Channel integrations retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel integrations retrieved successfully", integrations, paginationResponse)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) ActivateChannelIntegration(c *gin.Context) {
	org_id := c.Param("org_id")
	channel_id := c.Param("channel_id")
	integration_id := c.Param("integration_id")
	req := models.ActivateChannelIntegration{}

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to decode channel id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to bind request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to bind request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"channel_id":      channel_id,
		"integration_id":  integration_id,
	}

	err := integrations.ActivateChannelIntegration(ids, req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to activate channel integration")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Channel integration activated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel integration activated successfully", nil)
	c.JSON(http.StatusOK, rd)
}


func (base *Controller) DeactivateChannelIntegration(c *gin.Context){
	org_id := c.Param("org_id")
	channel_id := c.Param("channel_id")
	integration_id := c.Param("integration_id")
	req := models.DeactivateChannelIntegration{}

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel id format", "failed to decode channel id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Failed to bind request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to bind request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"channel_id":      channel_id,
		"integration_id":  integration_id,
	}

	err := integrations.DeactivateChannelIntegration(ids, req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to deactivate channel integration")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Channel integration deactivated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Channel integration deactivated successfully", nil)
	c.JSON(http.StatusOK, rd)
}