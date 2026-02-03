package models

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type BlogCategory struct {
	ID        string         `gorm:"type:uuid;primary_key" json:"id"`
	Name      string         `gorm:"not null;unique" json:"name"`
	CreatedAt time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type BlogCategoryCreateReq struct {
	Name string `json:"name" validate:"required"`
}

func (b *BlogCategory) CreateBlogCategory(db *gorm.DB, name string) error {
	var (
		blogCategory BlogCategory
		err          error
	)
	exists := postgresql.CheckExists(db, &blogCategory, "name = ?", name)

	if exists {
		return errors.New("blog category already exists")
	}

	err = postgresql.CreateOneRecord(db, &b)

	if err != nil {
		return err
	}

	return nil
}

func (b *BlogCategory) GetBlogCategories(db *gorm.DB, c *gin.Context) ([]BlogCategory, postgresql.PaginationResponse, error) {
	var blogCategory []BlogCategory

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&blogCategory,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return blogCategory, paginationResponse, nil
}

func (b *BlogCategory) GetBlogCategoryById(db *gorm.DB) error {
	err := postgresql.SelectFirstFromDb(db, &b)

	if err != nil {
		return err
	}

	return nil
}

func (b *BlogCategory) Delete(db *gorm.DB) error {
	err := postgresql.DeleteRecordFromDb(db, &b)

	if err != nil {
		return err
	}

	return nil
}
