package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/testimonial"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Testimonial(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	controller := testimonial.Controller{Db: db, Logger: logger, Validator: validator, ExtReq: extReq}

	testimonialAuthURL := r.Group(fmt.Sprintf("%v", ApiVersion), middleware.Authorize(db.Postgresql))
	testimonialURL := r.Group(fmt.Sprintf("%v", ApiVersion))
	{
		testimonialAuthURL.POST("/testimonials", controller.Create)
	}

	{
		testimonialURL.GET("/testimonials", controller.GetTestimonials)
	}
	return r
}
