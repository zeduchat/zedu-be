package test_helpcenter

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/helpcenter"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func initialise(currUUID string, t *testing.T, r *gin.Engine, user auth.Controller, status bool) string {
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	tst.SignupUser(t, r, user, userSignUpData, status)

	token := tst.GetLoginToken(t, r, user, loginData)

	return token
}

func SetupHelpCenterTestRouter() (*gin.Engine, *helpcenter.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	helpcenterController := &helpcenter.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupHelpCenterRoutes(r, helpcenterController)
	return r, helpcenterController
}

func SetupHelpCenterRoutes(r *gin.Engine, helpcenterController *helpcenter.Controller) {
	helpcenterAdminUrl := r.Group("/api/v1", middleware.Authorize(helpcenterController.Db.Postgresql))
	helpcenterUrl := r.Group("/api/v1")

	helpcenterAdminUrl.POST("/help-center/categories", helpcenterController.CreateHelpCenterCategory)
	helpcenterAdminUrl.POST("/help-center/articles/:category-id", helpcenterController.CreateHelpCenterArticle)

	helpcenterUrl.GET("/help-center/categories", helpcenterController.GetAllCategories)
	helpcenterUrl.GET("/help-center/articles/categories/:category-id", helpcenterController.GetArticlesByCategoryID)
	helpcenterUrl.GET("/help-center/articles/:id", helpcenterController.GetArticleByID)
}
