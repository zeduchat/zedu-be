package test_newsletter

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/newsletter"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupNewsletterTestRouter() (*gin.Engine, *newsletter.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	newsletterController := &newsletter.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupNewsletterRoutes(r, newsletterController)
	return r, newsletterController
}

func SetupNewsletterRoutes(r *gin.Engine, newsletterController *newsletter.Controller) {
	newsletterUrl := r.Group("/api/v1")

	newsletterUrl.POST("/newsletter", newsletterController.SubscribeNewsLetter)

}
