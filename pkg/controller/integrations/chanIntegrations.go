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

	integrations, paginationResponse, code, err := integrations.GetOrganisationChannelIntegrations(base.Db.Postgresql, channel_id, org_id, c, base.ExtReq)

	if err != nil {
		base.Logger.Error("Failed to get channel integrations")
		rd := utility.BuildErrorResponse(code, "error", "Failed to get channel integrations", err, nil)
		c.JSON(code, rd)
		return
	}


	base.Logger.Info("Channel integrations retrieved successfully")
	rd := utility.BuildSuccessResponse(code, "Channel integrations retrieved successfully", integrations, paginationResponse)
	c.JSON(code, rd)
}

func (base *Controller) ActivateDeactivateChannelIntegration(c *gin.Context) {
	org_id := c.Param("org_id")
	channel_id := c.Param("channel_id")
	integration_id := c.Param("integration_id")
	req := models.ActivateChannelIntegration{}

	var msg string

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

	if req.Status {
		msg = "Channel integration activated successfully"
	} else {
		msg = "Channel integration deactivated successfully"
	}

	base.Logger.Info(msg)
	rd := utility.BuildSuccessResponse(http.StatusOK, msg, nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) IntegrationChannels(c *gin.Context) {
	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"integration_id":  integration_id,
	}

	res, err := integrations.IntegrationChannels(ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to activate channel integration", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration channels fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration channels fetched successfully", res)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) CheckIntegrationIsActive(c *gin.Context) {
	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(integration_id); err != nil {
		base.Logger.Error("invalid integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"organisation_id": org_id,
		"integration_id":  integration_id,
	}

	res, err := integrations.CheckIntegrationIsActive(ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to fetch integration status", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration status fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration status fetched successfully", res)
	c.JSON(http.StatusOK, rd)
}
