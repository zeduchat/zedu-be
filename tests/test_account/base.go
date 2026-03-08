package test_account

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/pkg/controller/account"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupAccountTestRouter() (*gin.Engine, *account.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	accountController := &account.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	r := gin.Default()
	SetupAccountRoutes(r, accountController)
	return r, accountController
}

func SetupAccountRoutes(r *gin.Engine, accountController *account.Controller) {
	accountUrl := r.Group("/api/v1/account",
		middleware.Authorize(accountController.Db.Postgresql))
	{
		accountUrl.POST("/delete-account", accountController.CreateAccountDeletionRequest)
	}
}
