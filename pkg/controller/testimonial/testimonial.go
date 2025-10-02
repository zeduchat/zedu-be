package testimonial

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	service "github.com/hngprojects/telex_be/services/testimonial"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Logger    *utility.Logger
	Validator *validator.Validate
	ExtReq    request.ExternalRequest
}

func (base *Controller) Create(c *gin.Context) {
	var req models.TestimonialReq

	if err := c.ShouldBind(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		if err.Error() == "user claims not found" {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to create testimonial", nil)
			c.JSON(http.StatusNotFound, rd)
			return
		}
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), "failed to create testimonial", nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	userId := userID.(string)

	testimonial, err := service.CreateTestimonial(base.Db.Postgresql, req, userId)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "failed to create testimonial", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("testimonial created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "testimonial created successfully", testimonial)
	c.JSON(http.StatusCreated, rd)

}

func (base *Controller) GetTestimonials(c *gin.Context) {
	testimonials, paginationResponse, err := service.GetTestimonials(base.Db.Postgresql, c)

	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusNotFound, "error", "failed to fetch testimonials", err, nil)
		c.JSON(http.StatusNotFound, rd)
		return
	}

	paginationData := map[string]any{
		"current_page": paginationResponse.CurrentPage,
		"total_pages":  paginationResponse.TotalPagesCount,
		"page_size":    paginationResponse.PageCount,
		"total_items":  paginationResponse.TotalItems,
	}

	base.Logger.Info("testimonials retrieved successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "testimonials retrieved successfully", testimonials, paginationData)
	c.JSON(http.StatusOK, rd)

}
