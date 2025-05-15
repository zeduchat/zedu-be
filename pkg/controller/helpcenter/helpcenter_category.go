package helpcenter

import (
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/helpcenter"
	"github.com/hngprojects/telex_be/utility"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (base *Controller) CreateHelpCenterCategory(c *gin.Context) {
	var req models.CreateHelpCenterCategory

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.Name = utility.CleanStringInput(req.Name)
	req.Description = utility.CleanStringInput(req.Description)

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Input validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, err := service.CreateHelpCenterCategory(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to add Article", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Category added successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Category added successfully", respData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetAllCategories(c *gin.Context) {
	topics, paginationResponse, err := service.GetPaginatedCategories(c, base.Db.Postgresql)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Categories not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch Categories", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	paginationData := map[string]interface{}{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(topics),
	}

	base.Logger.Info("Categories retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Categories retrieved successfully.", topics, paginationData)
	c.JSON(http.StatusOK, rd)
}
