package integrations

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/integrations"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func GetAllIntegrationApp(c *gin.Context, org_id string, db *gorm.DB) ([]models.Integrations, error) {
	integrations := models.Integrations{}
	intApps, err := integrations.GetAllIntegrationApp(db, org_id, c)

	if err != nil {
		return nil, err
	}
	return intApps, nil
}

func (base *Controller) GetAllIntegrationApp(c *gin.Context) {
	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	integrations, err := integrations.GetAllIntegrationApp(c, org_id, base.Db.Postgresql)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			base.Logger.Error("integrations not found", err)
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "integrations not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			base.Logger.Error("Failed to fetch integrations", err)
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch integrations", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("integrations retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "integrations retrieved successfully.", integrations)
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
		base.Logger.Error("Failed to update integration app", err)
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
		base.Logger.Error("Failed to delete integration app", err)
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
		base.Logger.Error("Failed to set integration app status", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to set integration app status", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Integration app status set successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Integration app status set successfully", nil)
	c.JSON(http.StatusOK, rd)
}

func fetchJSONFromURL(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var jsonSchema map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&jsonSchema)
	if err != nil {
		return nil, err
	}
	return jsonSchema, nil
}
