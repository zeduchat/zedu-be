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

func CreateBlog(req models.BlogCreateReq, db *gorm.DB, userId string) error {
	var blogCategory models.BlogCategory
	blogCategory.ID = req.CategoryID

	err := blogCategory.GetBlogCategoryById(db)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("blog category not found")
		}
		return err
	}

	req.Title = utility.CleanStringInput(req.Title)
	req.Content = utility.CleanStringInput(req.Content)

	blog := models.Blog{
		ID:         utility.GenerateUUID(),
		Title:      req.Title,
		Content:    req.Content,
		CategoryID: req.CategoryID,
		ImageURL:   req.ImageURL,
		AuthorID:   userId,
	}

	err = blog.Create(db)

	if err != nil {
		return err
	}

	return nil
}

func GetBlogs(db *gorm.DB, c *gin.Context, categoryID string, searchQuery string) ([]models.Blog, postgresql.PaginationResponse, error) {
	var (
		blog models.Blog
	)
	searchQuery = strings.Trim(searchQuery, `"'`)
	categoryID = strings.Trim(categoryID, `"'`)
	blogs, paginationResponse, err := blog.GetBlogs(db, c, categoryID, searchQuery)
	if err != nil {
		return nil, paginationResponse, err
	}

	return blogs, paginationResponse, nil

}

func GetBlogById(blogId string, db *gorm.DB) (models.Blog, error) {
	var (
		blog         models.Blog
		blogCategory models.BlogCategory
	)

	blog.ID = blogId

	err := blog.GetBlogById(db)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return blog, errors.New("blog not found")
		}
		return blog, err
	}

	blogCategory.ID = blog.CategoryID
	err = blogCategory.GetBlogCategoryById(db)
	
	if err != nil {
		return blog, err
	}

	blog.Category = &blogCategory

	return blog, nil
}

func DeleteBlog(blogId string, userId string, db *gorm.DB) error {
	var blog models.Blog
	blog.ID = blogId

	err := blog.GetBlogById(db)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("blog not found")
		}
		return err
	}

	if blog.AuthorID != userId {
		return errors.New("user not authorised to delete blog")
	}

	return blog.Delete(db)
}
