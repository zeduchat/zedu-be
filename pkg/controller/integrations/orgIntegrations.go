package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateIntegrationApp(c *gin.Context) {
	var req models.Integrations
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Input validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Input Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	integration, err := integrations.CreateIntegrationApp(req, org_id, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to create integration app")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to create integration app", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Integration app created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Integration app created successfully", integration)
	c.JSON(http.StatusCreated, rd)
}

func GetAllIntegrationApp(c *gin.Context, db *gorm.DB) ([]models.Integrations, postgresql.PaginationResponse, error) {
	integrations := models.Integrations{}
	intApps, paginationResponse, err := integrations.GetAllIntegrationApp(db, c)

	if err != nil {
		return nil, paginationResponse, err
	}

	return intApps, paginationResponse, nil
}

func (base *Controller) GetAllIntegrationApp(c *gin.Context) {
	integrations, paginationResponse, err := integrations.GetAllIntegrationApp(c, base.Db.Postgresql)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "integrations not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch integrations", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
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

func (base *Controller) UpdateIntegrationApp(c *gin.Context) {
	var req models.UpdateIntegration
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

	err := base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Error("Input validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Input validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	updatedIntegration, err := integrations.UpdateIntegrationApp(req, ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to update integration app")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to update integration app", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Integrations updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integrations updated successfully", updatedIntegration)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteIntegrationApp(c *gin.Context) {
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
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	err := integrations.DeleteIntegrationApp(ids, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to delete integration app")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to delete integration app", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Integration app deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration app deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) SetIntegrationActiveStatus(c *gin.Context) {
	org_id := c.Param("org_id")
	integration_id := c.Param("integration_id")
	status := c.Query("status")

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

	if status != "active" && status != "inactive" {
		base.Logger.Error("invalid status value")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid status value", "status value must be 'active' or 'inactive'", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	ids := map[string]string{
		"org_id":         org_id,
		"integration_id": integration_id,
	}

	err := integrations.SetIntegrationAppStatus(ids, status, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("Failed to set integration app status")
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to set integration app status", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Integration app status set successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration app status set successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) SlackIntegrationApp(c *gin.Context) {
	var req models.Integrations

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.Name = utility.CleanStringInput(req.Name)

	err := base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Input validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, err := integrations.SlackIntegrationApp(req, base.Db.Postgresql)
	if err != nil {
		if err.Error() == "integration app already exists" {
			rd := utility.BuildErrorResponse(http.StatusConflict, "error", err.Error(), err, nil)
			c.JSON(http.StatusConflict, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Application added successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Application added successfully", respData)
	c.JSON(http.StatusCreated, rd)
}
