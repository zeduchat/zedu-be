package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) AddIntegrationSlashCommand(c *gin.Context) {
	var (
		req models.AddSlashCommandRequest
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

	response, err := integrations.AddIntegrationsSlashCommand(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to add integration slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to add integration slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration setting added successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration setting added successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetIntegrationSlashCommands(c *gin.Context) {
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

	response, err := integrations.GetIntegrationSlashCommands(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to get integration slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get integration slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration slashcommands retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration slashcommands retrieved successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateIntegrationSlashCommand(c *gin.Context) {
	var (
		req            models.UpdateSlashCommandRequest
		org_id         = c.Param("org_id")
		integration_id = c.Param("integration_id")
		command_id     = c.Param("command_id")
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

	if _, err := uuid.Parse(command_id); err != nil {
		base.Logger.Error("invalid command id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid command id format", "failed to decode integration id", nil)
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
		"command_id":     command_id,
	}

	response, err := integrations.UpdateIntegrationSlashCommand(base.Db.Postgresql, ids, req)
	if err != nil {
		base.Logger.Error("Failed to update integration slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to update integration slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration setting updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration setting updated successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteIntegrationSlashCommand(c *gin.Context) {
	var (
		org_id         = c.Param("org_id")
		integration_id = c.Param("integration_id")
		command_id     = c.Param("command_id")
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

	if _, err := uuid.Parse(command_id); err != nil {
		base.Logger.Error("invalid command id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid command id format", "failed to decode command id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"integration_id": integration_id,
		"command_id":     command_id,
	}

	err := integrations.DeleteIntegrationSlashCommand(base.Db.Postgresql, ids)
	if err != nil {
		base.Logger.Error("Failed to delete integration slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to delete integration slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration slashcommands deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration slashcommands deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetAllOrgSlashCommands(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, err := integrations.GetAllOrgSlashCommands(base.Db.Postgresql, org_id)
	if err != nil {
		base.Logger.Error("Failed to get all organisation slashcommands", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get all organisation slashcommands", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Organisation slashcommands retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Organisation slashcommands retrieved successfully", response)
	c.JSON(http.StatusOK, rd)
}
