package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/blog"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Blog(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	blogs := blog.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	blogsAdminUrl := r.Group(fmt.Sprintf("%v", ApiVersion), middleware.Authorize(db.Postgresql))
	blogsUrl := r.Group(fmt.Sprintf("%v", ApiVersion))

	{
		blogsAdminUrl.POST("/blogs", blogs.CreateBlog)
		blogsAdminUrl.POST("/blog_categories", blogs.CreateBlogCategory)
		blogsAdminUrl.DELETE("/blogs/:id", blogs.DeleteBlog)
	}

	{
		blogsUrl.GET("/blogs", blogs.GetBlogs)
		blogsUrl.GET("/blog_categories", blogs.GetBlogCategories)
		blogsUrl.GET("/blog_categories/:id", blogs.GetBlogCategoryById)
		blogsUrl.GET("/blogs/:id", blogs.GetBlogById)
	}

	return r
}
