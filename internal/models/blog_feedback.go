package models

import (
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type BlogFeedback struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	BlogID    string    `gorm:"type:uuid;not null" json:"blog_id"`
	Blog      *Blog     `gorm:"foreignKey:BlogID" json:"blog,omitempty"`
	Feedback  bool      `gorm:"not null" json:"feedback"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type BlogFeedbackReq struct {
	BlogID   string `json:"blog_id" validate:"required"`
	Feedback bool   `json:"feedback" validate:"required"`
}

func (b *BlogFeedback) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &b)

	if err != nil {
		return err
	}

	return nil
}
