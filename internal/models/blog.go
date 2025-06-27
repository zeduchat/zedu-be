package models

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Blog struct {
	ID           string         `gorm:"type:uuid;primary_key" json:"id"`
	Title        string         `gorm:"not null" json:"title"`
	Content      string         `gorm:"type:text" json:"content"`
	HTMLContent  string         `gorm:"type:text" json:"html_content"`
	Summary      string         `gorm:"type:text" json:"summary"`
	ImageURL     string         `gorm:"type:text" json:"image_url,omitempty"`
	PreviewImage string         `gorm:"type:text" json:"preview_image,omitempty"`
	CategoryID   string         `gorm:"type:uuid;not null" json:"category_id"`
	Category     *BlogCategory  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	AuthorID     string         `gorm:"type:uuid;not null" json:"author_id"`
	AuthorName   string         `gorm:"type:text" json:"author_name"`
	AuthorAvatar string         `gorm:"type:text" json:"author_avatar"`
	Author       *User          `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	PublishedAt  string         `gorm:"type:text" json:"published_at"`
	CreatedAt    time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type BlogCreateReq struct {
	CategoryID string `json:"category_id"`
	Content    string `json:"content"`
}

func (b *Blog) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &b)

	if err != nil {
		return err
	}

	return nil
}

func (b *Blog) GetBlogs(db *gorm.DB, c *gin.Context, categoryID string, searchQuery string) ([]Blog, postgresql.PaginationResponse, error) {
	var blog []Blog

	pagination := postgresql.GetPagination(c)

	queryConditions := ""
	queryArgs := []any{}

	if categoryID != "" {
		queryConditions += "category_id = ?"
		queryArgs = append(queryArgs, categoryID)
	}

	if searchQuery != "" {
		if queryConditions != "" {
			queryConditions += " AND "
		}
		queryConditions += "title LIKE ? OR content LIKE ?"
		searchPattern := "%" + searchQuery + "%"
		queryArgs = append(queryArgs, searchPattern, searchPattern)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&blog,
		queryConditions,
		queryArgs...,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return blog, paginationResponse, nil
}

func (b *Blog) GetBlogById(db *gorm.DB) error {
	err := postgresql.SelectFirstFromDb(db, &b)

	if err != nil {
		return err
	}

	return nil
}

func (b *Blog) Delete(db *gorm.DB) error {
	err := postgresql.DeleteRecordFromDb(db, &b)

	if err != nil {
		return err
	}

	return nil
}
