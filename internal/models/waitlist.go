package models

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type Waitlist struct {
	ID        string         `gorm:"primaryKey;type:uuid" json:"id"`
	Email     string         `gorm:"unique;not null;column:email" json:"email" validate:"required,email"`
	Name      string         `gorm:"column:name" json:"name"`
	CreatedAt time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type WaitlistRequest struct {
	Email string `json:"email" validate:"required"`
	Name  string `json:"name" validate:"required"`
}

func (n *Waitlist) CreateWaitlist(db *gorm.DB, req WaitlistRequest) error {

	if postgresql.CheckExists(db, &n, "email = ?", req.Email) {
		return errors.New("email already subscribed")
	}

	n.ID = utility.GenerateUUID()
	n.Email = req.Email
	n.Name = req.Name

	err := postgresql.CreateOneRecord(db, &n)

	if err != nil {
		return err
	}

	return nil
}
