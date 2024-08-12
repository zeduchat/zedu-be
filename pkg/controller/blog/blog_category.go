package blog

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/blog"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreateBlogCategory(c *gin.Context) {
	var req models.BlogCategoryCreateReq

	if err := c.ShouldBind(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	category, err := service.CreateBlogCategory(req, base.Db.Postgresql)

	if err != nil {
		if err.Error() == "blog category already exists" {
			rd := utility.BuildErrorResponse(http.StatusConflict, "error", err.Error(), "failed to create blog category", nil)
			c.JSON(http.StatusConflict, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("blog category created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "blog category created successfully", category)

	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetBlogCategories(c *gin.Context) {
	blogCategories, paginationResponse, err := service.GetBlogCategories(base.Db.Postgresql, c)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "failed to fetch blog categories", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	paginationData := map[string]interface{}{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(blogCategories),
	}

	base.Logger.Info("blog categories retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "blog categories retrieved successfully", blogCategories, paginationData)
	c.JSON(http.StatusOK, rd)
}

