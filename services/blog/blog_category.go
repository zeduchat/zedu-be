package blog

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateBlogCategory(req models.BlogCategoryCreateReq, db *gorm.DB) (models.BlogCategory, error) {
	req.Name = utility.CleanStringInput(req.Name)

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

func GetBlogCategoryById(id string, db *gorm.DB) (models.BlogCategory, error) {
	var blogCategory models.BlogCategory

	blogCategory.ID = id
	err := blogCategory.GetBlogCategoryById(db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.BlogCategory{}, errors.New("blog category not found")
		}
		return models.BlogCategory{}, err
	}

	return blogCategory, nil

}

func DeleteBlogCategory(categoryId string, db *gorm.DB) error {
	var blogCategory models.BlogCategory
	blogCategory.ID = categoryId

	err := blogCategory.GetBlogCategoryById(db)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("blog category not found")
		}
		return err
	}

	return blogCategory.Delete(db)
}
