package models

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Testimonial struct {
	ID          string         `gorm:"type:uuid;primary_key" json:"id"`
	UserID      string         `gorm:"type:uuid;not null" json:"user_id"`
	CompanyName string         `gorm:"type:varchar(255);not null" json:"company_name"`
	Name        string         `gorm:"type:text;not null" json:"name"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	ImageURL    string         `gorm:"type:varchar(255)" json:"image_url,omitempty"`
	CreatedAt   time.Time      `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type TestimonialReq struct {
	CompanyName string `json:"company_name" validate:"required"`
	Content     string `json:"content" validate:"required"`
}

func (t *Testimonial) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &t)

	if err != nil {
		return err
	}

	return nil
}

func (b *Testimonial) GetAllTestimonials(db *gorm.DB, c *gin.Context) ([]Testimonial, postgresql.PaginationResponse, error) {
	var testimonial []Testimonial

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&testimonial,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return testimonial, paginationResponse, nil
}
