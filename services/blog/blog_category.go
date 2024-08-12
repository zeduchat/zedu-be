package blog

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateBlogCategory(req models.BlogCategoryCreateReq, db *gorm.DB) (models.BlogCategory, error) {
	blogCategory := models.BlogCategory{
		ID:   utility.GenerateUUID(),
		Name: strings.ToLower(req.Name),
	}

	name := strings.ToLower(req.Name)

	err := blogCategory.CreateBlogCategory(db, name)

	if err != nil {
		return models.BlogCategory{}, err
	}

	return blogCategory, nil
}

func GetBlogCategories(db *gorm.DB, c *gin.Context) ([]models.BlogCategory, postgresql.PaginationResponse, error) {
	var (
		blogCategory models.BlogCategory
	)

	blogCategories, paginationResponse, err := blogCategory.GetBlogCategories(db, c)
	if err != nil {
		return nil, paginationResponse, err
	}

	return blogCategories, paginationResponse, nil
}
