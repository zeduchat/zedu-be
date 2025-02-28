package integrations

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) FetchOutputIntegrations(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	integrations, err := integrations.GetActiveOutputIntegrations(base.Db.Postgresql, org_id)
	if err != nil {
		base.Logger.Error("Failed to get out putintegrations")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to get output integrations", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	base.Logger.Info("output integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "output integrations retrieved successfully.", integrations)
	c.JSON(http.StatusOK, rd)
}

// Fetch System Integration without org details (Unauthorized)
func (base *Controller) GetSystemIntegrationApps(c *gin.Context) {

	integrations, paginationResponse, err, code := integrations.GetSystemIntegrationApps(c, base.Db.Postgresql, base.ExtReq)
	if err != nil {
		fmt.Println(err)
		base.Logger.Error("Failed to fetch integrations", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch integrations", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	paginationData := map[string]interface{}{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(integrations),
	}

	base.Logger.Info("integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "integrations retrieved successfully.", integrations, paginationData)
	c.JSON(http.StatusOK, rd)
}

// Get Single Integration App
func (base *Controller) GetSystemIntegrationApp(c *gin.Context) {

	int_id := c.Param("integration_id")

	if _, err := uuid.Parse(int_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid integration id format", "failed to decode integration id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	integration, err, code := integrations.GetSystemIntegrationApp(c, base.Db.Postgresql, int_id, base.ExtReq)
	if err != nil {
		base.Logger.Error("Failed to fetch integrations", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to fetch integrations", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "integrations retrieved successfully.", integration)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) TriggerTick(c *gin.Context) {
	var req models.TriggerTickRequest

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

	response, code, err := integrations.TriggerTick(base.Db, base.Logger, req)
	if err != nil {
		base.Logger.Error("Failed to trigger tick", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to trigger tick", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("tick called successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "tick called successfully", response)
	c.JSON(http.StatusOK, rd)
}
