package models

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type OptIn struct {
	ID        string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	FirstName string    `gorm:"column:first_name; type:varchar(255); not null" json:"first_name"`
	LastName  string    `gorm:"column:last_name; type:varchar(255); not null" json:"last_name"`
	Email     string    `gorm:"column:email; type:varchar(255); not null" json:"email"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type CreateOptIn struct {
	FirstName string `json:"first_name" validate:"required,min=2,max=255"`
	LastName  string `json:"last_name" validate:"required,min=2,max=255"`
	Email     string `json:"email" validate:"required,email"`
}

func (o *OptIn) CreateOptInRecord(db *gorm.DB, email string) (int, error) {

	if postgresql.CheckExists(db, &OptIn{}, "email = ?", email) {
		return http.StatusConflict, errors.New("email already opted in, please use a different email or stay tuned")
	}

	err := postgresql.CreateOneRecord(db, &o)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil
}

func (o *OptIn) GetAllOptInRecords(db *gorm.DB, c *gin.Context) ([]OptIn, postgresql.PaginationResponse, error) {
	var optIns []OptIn

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&optIns,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return optIns, paginationResponse, nil
}
