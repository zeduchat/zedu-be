package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AddIntegrationSetting(c *gin.Context) {
	var (
		req models.AddIntegrationSettingsRequest
	)

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

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	err := integrations.AddIntegrationSettings(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to add integration settings", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to add integration settings", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration setting added successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration setting added successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetIntegrationSettings(c *gin.Context) {
	var (
		org_id         = c.Param("org_id")
		integration_id = c.Param("integration_id")
	)

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
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	setting, err := integrations.GetIntegrationSetting(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to get organisation integration setting", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get organisation integration setting", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration setting retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation Integration setting retrieved successfully", setting)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateChannelIntegrationSetting(c *gin.Context) {
	var (
		req            models.UpdateIntegrationSettingsRequest
		org_id         = c.Param("org_id")
		integration_id = c.Param("integration_id")
		channel_id     = c.Param("channel_id")
		setting_id     = c.Param("setting_id")
	)

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

	if _, err := uuid.Parse(channel_id); err != nil {
		base.Logger.Error("invalid channel integration id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid channel integration id format", "failed to decode channel integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(setting_id); err != nil {
		base.Logger.Error("invalid setting id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid setting id format", "failed to decode setting id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"integration_id": integration_id,
		"channel_id":     channel_id,
		"setting_id":     setting_id,
	}

	err := integrations.UpdateChannelIntegrationSettings(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to update integration setting", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update integration setting", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration setting updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration setting updated successfully", nil)
	c.JSON(http.StatusOK, rd)
}
