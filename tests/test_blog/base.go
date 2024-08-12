package test_blog

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/blog"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupBlogTestRouter() (*gin.Engine, *blog.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	blogController := &blog.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupBlogRoutes(r, blogController)
	return r, blogController
}

func SetupBlogRoutes(r *gin.Engine, blogController *blog.Controller) {
	blogAdminUrl := r.Group("/api/v1", middleware.Authorize(blogController.Db.Postgresql))
	blogUrl := r.Group("/api/v1")

	blogAdminUrl.POST("/blogs", blogController.CreateBlog)
	blogAdminUrl.POST("/blog_categories", blogController.CreateBlogCategory)

	blogUrl.GET("/blogs", blogController.GetBlogs)
	blogUrl.GET("/blog_categories", blogController.GetBlogCategories)
	blogUrl.GET("/blogs/:id", blogController.GetBlogById)
}
