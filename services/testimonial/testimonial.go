package testimonial

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateTestimonial(db *gorm.DB, req models.TestimonialReq, userId string) (*models.Testimonial, error) {
	var user models.User
	user, _ = user.GetUserWithProfile(db, userId)

	req.Content = utility.CleanStringInput(req.Content)
	testimonial := &models.Testimonial{
		ID:          utility.GenerateUUID(),
		UserID:      userId,
		CompanyName: req.CompanyName,
		Name:        user.Name,
		Content:     req.Content,
		ImageURL:    user.Profile.AvatarURL,
	}

	err := testimonial.Create(db)

	if err != nil {
		return nil, err
	}

	return testimonial, nil

}

func GetTestimonials(db *gorm.DB, c *gin.Context) ([]models.Testimonial, postgresql.PaginationResponse, error) {
	var testimonial models.Testimonial

	testimonials, paginationResponse, err := testimonial.GetAllTestimonials(db, c)

	if err != nil {
		return nil, paginationResponse, err
	}

	return testimonials, paginationResponse, nil
}
