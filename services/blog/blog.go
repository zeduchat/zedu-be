package blog

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/blog_utility"
	"gorm.io/gorm"
)

func CreateBlog(req models.BlogCreateReq, db *gorm.DB, adminId string) error {
	var blogCategory models.BlogCategory
	blogCategory.ID = req.CategoryID

	err := blogCategory.GetBlogCategoryById(db)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("blog category not found")
		}
		return err
	}

	var admin models.Admin
	if err := db.First(&admin, "id = ?", adminId).Error; err != nil {
		return errors.New("admin not found")
	}

	// Platform admins are created from existing Zedu users. Resolve that user
	// by the shared email so the existing Blog.Author relationship remains
	// valid while authorization is enforced with the admin session.
	var user models.User
	if err := db.Where("LOWER(email) = ?", strings.ToLower(admin.Email)).First(&user).Error; err != nil {
		return errors.New("admin user profile not found")
	}
	user, _ = user.GetUserWithProfile(db, user.ID)

	frontMatter, markdownContent, err := blog_utility.ParseFrontMatter(req.Content)
	if err != nil {
		return err
	}

	htmlContent := blog_utility.ConvertMarkdownToHTML(markdownContent)

	blog := models.Blog{
		ID:           utility.GenerateUUID(),
		Title:        frontMatter.Title,
		Content:      req.Content,
		HTMLContent:  htmlContent,
		Summary:      frontMatter.Summary,
		ImageURL:     frontMatter.Image,
		PreviewImage: frontMatter.PreviewImage,
		CategoryID:   req.CategoryID,
		AuthorID:     user.ID,
		AuthorName:   user.Name,
		AuthorAvatar: user.Profile.AvatarURL,
		PublishedAt:  frontMatter.PublishedAt,
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

func DeleteBlog(blogId string, db *gorm.DB) error {
	var blog models.Blog
	blog.ID = blogId

	err := blog.GetBlogById(db)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("blog not found")
		}
		return err
	}

	return blog.Delete(db)
}
