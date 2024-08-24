package test_testimonial

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/testimonial"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupTestimonialTestRouter() (*gin.Engine, *testimonial.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	testimonialController := &testimonial.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupTestimonialRoutes(r, testimonialController)
	return r, testimonialController
}

func SetupTestimonialRoutes(r *gin.Engine, testimonialController *testimonial.Controller) {
	r.POST(
		"/api/v1/testimonials",
		middleware.Authorize(testimonialController.Db.Postgresql),
		testimonialController.Create,
	)

	r.GET(
		"/api/v1/testimonials",
		testimonialController.GetTestimonials,
	)
}
