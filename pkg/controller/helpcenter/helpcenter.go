package helpcenter

import (
	"net/http"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/helpcenter"
	"github.com/hngprojects/telex_be/utility"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateHelpCenterArticle(c *gin.Context) {
	category_id := c.Param("category-id")
	var req models.CreateHelpCenterArticle

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req.Title = utility.CleanStringInput(req.Title)
	req.Content = utility.CleanStringInput(req.Content)
	req.CategoryID = category_id

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Input validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, err := service.CreateHelpCenterArticle(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("Article added successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Article added successfully", respData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetArticlesByCategoryID(c *gin.Context) {
	id := c.Param("category-id")

	if _, err := uuid.Parse(id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid ID format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	articles, paginationResponse, err := service.GetArticlesByCategoryID(c, base.Db.Postgresql, id)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Category not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch Category", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(articles),
	}

	base.Logger.Info("Articles retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Articles retrieved successfully.", articles, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetArticleByID(c *gin.Context) {
	id := c.Param("id")

	if _, err := uuid.Parse(id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid ID format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, err := service.GetArticleByID(base.Db.Postgresql, id)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Topic not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to fetch topic", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("Article retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Article retrieved successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) SearchHelpCenterArticles(c *gin.Context) {
	query := c.Query("title")

	if query == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Query parameter is required", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	topics, paginationResponse, err := service.SearchHelpCenterArticles(c, base.Db.Postgresql, query)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "No topics found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to search topics", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(topics),
	}

	base.Logger.Info("Articles retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Articles retrieved successfully.", topics, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateHelpCenterByID(c *gin.Context) {
	var req models.HelpCenterArticle
	id := c.Param("id")

	if _, err := uuid.Parse(id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid ID format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	result, err := service.UpdateArticleByID(base.Db.Postgresql, req, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Article not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to update Article", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("Article updated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Article updated successfully", result)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteTopicByID(c *gin.Context) {
	id := c.Param("id")

	if _, err := uuid.Parse(id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid ID format", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err := service.DeleteArticleByID(base.Db.Postgresql, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "Article not found", err, nil)
			c.JSON(http.StatusNotFound, rd)
		} else {
			rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to delete Article", err, nil)
			c.JSON(http.StatusInternalServerError, rd)
		}
		return
	}

	base.Logger.Info("Article deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusNoContent, "", nil)
	c.JSON(http.StatusNoContent, rd)

}
