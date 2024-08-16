package test_contact

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/pkg/controller/contact"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupContactTestRouter() (*gin.Engine, *contact.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	contactController := &contact.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	r := gin.Default()
	SetupContactRoutes(r, contactController)
	return r, contactController
}

func SetupContactRoutes(r *gin.Engine, contactController *contact.Controller) {
	r.POST("/api/v1/contact", contactController.AddToContactUs)
}
