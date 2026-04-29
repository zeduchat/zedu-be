package zedu

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetResources(c *gin.Context) {
	var resourcePages []models.ZeduResourcePage
	if err := base.Db.Postgresql.Find(&resourcePages).Error; err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch resource pages", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	var guides []models.ZeduGuide
	if err := base.Db.Postgresql.Find(&guides).Error; err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to fetch guides", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	data := map[string]any{
		"resource_pages": resourcePages,
		"guides":         guides,
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "resources retrieved successfully", data, nil)
	c.JSON(http.StatusOK, rd)
}
