package testoptin

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	optin "github.com/hngprojects/telex_be/pkg/controller/optIn"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupOptInTestRouter() (*gin.Engine, *optin.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	optinController := &optin.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupOptInRoutes(r, optinController)
	return r, optinController
}

func SetupOptInRoutes(r *gin.Engine, optinController *optin.Controller) {
	optInUrl := r.Group("/api/v1")
	optInUrl.POST("/optin", optinController.CreateOptInRecord)
}