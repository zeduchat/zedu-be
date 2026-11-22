package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type NewsLetter struct {
	ID        string         `gorm:"primaryKey;type:uuid" json:"id"`
	Email     string         `gorm:"unique;not null" json:"email" validate:"required,email"`
	CreatedAt time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Pending is true until the owner of the address clicks the link in the
	// confirmation email. Rows created before double opt-in stay false (confirmed).
	Pending bool `gorm:"column:pending;not null;default:false" json:"-"`
}

func (n *NewsLetter) BeforeCreate(tx *gorm.DB) (err error) {

	if n.ID == "" {
		n.ID = utility.GenerateUUID()
	}
	return
}

func (n *NewsLetter) CreateNewsLetter(db *gorm.DB, email string) error {

	if postgresql.CheckExists(db, &n, "email = ?", email) {
		return errors.New("email already subscribed")
	}

	err := postgresql.CreateOneRecord(db, &n)

	if err != nil {
		return err
	}

	return nil
}

type ConfirmNewsLetterRequest struct {
	Email string `json:"email" validate:"required,email"`
	Token string `json:"token" validate:"required"`
}

type UnsubscribeNewsLetterRequest struct {
	Email string `json:"email" validate:"required,email"`
	Token string `json:"token" validate:"required"`
}

// ConfirmNewsLetterByEmail marks a pending subscriber as confirmed. It reports
// whether a pending row was changed, so a repeated click does nothing.
func (n *NewsLetter) ConfirmNewsLetterByEmail(db *gorm.DB, email string) (bool, error) {
	tx := db.Model(&NewsLetter{}).Where("email = ? AND pending = ?", email, true).Update("pending", false)
	return tx.RowsAffected > 0, tx.Error
}

// DeleteNewsLetterByEmail removes the subscriber for good so that the address
// can subscribe again later without hitting the unique email constraint.
func (n *NewsLetter) DeleteNewsLetterByEmail(db *gorm.DB, email string) error {
	return postgresql.HardDeleteSpecificRecord(db, &NewsLetter{}, "email = ?", email)
}

func (n *NewsLetter) GetNewsLetterById(db *gorm.DB, ID string) (NewsLetter, error) {
	var newsletter NewsLetter

	err, nerr := postgresql.SelectOneFromDb(db, &newsletter, "id = ?", ID)
	if nerr != nil {
		return newsletter, err
	}
	return newsletter, nil
}

func (n *NewsLetter) GetDeletedNewsLetterById(db *gorm.DB, ID string) (NewsLetter, error) {
	var newsletter NewsLetter

	err := db.Unscoped().Where("id = ?", ID).First(&newsletter).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newsletter, fmt.Errorf("newsletter not found: %w", err)
		}
		return newsletter, fmt.Errorf("failed to retrieve newsletter: %w", err)
	}

	return newsletter, nil
}

func (n *NewsLetter) DeleteNewsLetter(db *gorm.DB) error {

	err := postgresql.DeleteRecordFromDb(db, &n)

	if err != nil {
		return err
	}

	return nil
}

func (n *NewsLetter) UpdateNewsLetter(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &n)
	return err
}

func (n *NewsLetter) FetchAllNewsLetter(db *gorm.DB, c *gin.Context) ([]NewsLetter, postgresql.PaginationResponse, error) {
	var newsLetters []NewsLetter

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&newsLetters,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return newsLetters, paginationResponse, nil
}

func (n *NewsLetter) FetchAllDeletedNewsLetter(db *gorm.DB, c *gin.Context) ([]NewsLetter, postgresql.PaginationResponse, error) {
	var newsLetters []NewsLetter

	pagination := postgresql.GetPagination(c)

	query := db.Unscoped().Where("deleted_at IS NOT NULL")

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&newsLetters,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return newsLetters, paginationResponse, nil
}
