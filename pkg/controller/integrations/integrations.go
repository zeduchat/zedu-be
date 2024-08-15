package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.Name = utility.CleanStringInput(req.Name)

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Input validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, err := integrations.CreateIntegrationApp(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to add Article", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Application added successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Application added successfully", respData)
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
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "No Job post not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch job post", err, nil)
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
	base.Logger.Info("Topics retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Topics retrieved successfully.", integrations, paginationData)
	c.JSON(http.StatusOK, rd)
}