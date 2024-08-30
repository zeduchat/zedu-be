package blog

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/blog"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateBlog(c *gin.Context) {
	categoryID := c.PostForm("category_id")
	if categoryID == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Category ID is required", "failed to create blog", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if _, err := uuid.Parse(categoryID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid category id format", "failed to create blog", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	file, err := c.FormFile("content")
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to get content file", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	contentFile, err := file.Open()
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to open content file", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	defer contentFile.Close()

	contentBytes, err := io.ReadAll(contentFile)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to read content file", err, nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	content := string(contentBytes)

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to create blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to create blog", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	err = service.CreateBlog(models.BlogCreateReq{
		CategoryID: categoryID,
		Content:    content,
	}, base.Db.Postgresql, userId)

	if err != nil {
		if err.Error() == "blog category not found" {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to create blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("blog created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "blog created successfully", nil)

	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetBlogs(c *gin.Context) {
	categoryID := c.Query("category")
	searchQuery := c.Query("search")

	blogs, paginationResponse, err := service.GetBlogs(base.Db.Postgresql, c, categoryID, searchQuery)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "failed to fetch blogs", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	paginationData := map[string]interface{}{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  len(blogs),
	}

	base.Logger.Info("blogs retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "blogs retrieved successfully", blogs, paginationData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetBlogById(c *gin.Context) {
	blogID := c.Param("id")

	if _, err := uuid.Parse(blogID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid blog id format", "failed to retrieve blog", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	blog, err := service.GetBlogById(blogID, base.Db.Postgresql)

	if err != nil {
		if err.Error() == "blog not found" {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to retrieve blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to retrieve blog", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("blog retrieved successfully.")
	rd := utility.BuildSuccessResponse(http.StatusOK, "blog retrieved successfully", blog)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteBlog(c *gin.Context) {
	blogID := c.Param("id")

	if _, err := uuid.Parse(blogID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid blog id format", "failed to delete blog", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to delete blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to delete blog", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	if err := service.DeleteBlog(blogID, userId, base.Db.Postgresql); err != nil {
		if err.Error() == "blog not found" {
			rd := utility.BuildErrorResponse(http.StatusNotFound, "error", err.Error(), "failed to delete blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		if err.Error() == "user not authorised to delete blog" {
			rd := utility.BuildErrorResponse(http.StatusForbidden, "error", err.Error(), "failed to delete blog", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "failed to delete blog", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("blog successfully deleted")
	rd := utility.BuildSuccessResponse(http.StatusNoContent, "", nil)
	c.JSON(http.StatusNoContent, rd)

}
