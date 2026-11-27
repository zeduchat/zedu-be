package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/helpcenter"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func HelpCenter(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	helpcenter := helpcenter.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	// Back-office only: every help-center write route belongs on this group.
	// The content is served publicly, so ordinary user tokens must not reach it.
	helpcenterAdminUrl := r.Group(fmt.Sprintf("%v", ApiVersion), middleware.AdminAuthorize(db.Postgresql))
	helpcenterUrl := r.Group(fmt.Sprintf("%v", ApiVersion))

	{
		helpcenterAdminUrl.POST("/help-center/categories", helpcenter.CreateHelpCenterCategory)
		helpcenterAdminUrl.POST("/help-center/articles/:category-id", helpcenter.CreateHelpCenterArticle)
	}

	{
		helpcenterUrl.GET("/help-center/categories", helpcenter.GetAllCategories)
		helpcenterUrl.GET("/help-center/articles/categories/:category-id", helpcenter.GetArticlesByCategoryID)
		helpcenterUrl.GET("/help-center/articles/search", helpcenter.SearchHelpCenterArticles)
		helpcenterUrl.GET("/help-center/articles/:id", helpcenter.GetArticleByID)
	}

	return r
}
