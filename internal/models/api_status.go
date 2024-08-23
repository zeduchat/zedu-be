package models

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type APIStatus struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	APIGroup       string    `gorm:"type:text;not null" json:"api_group"`
	Status         string    `gorm:"type:text;not null" json:"status"`
	LastChecked    time.Time `gorm:"not null" json:"last_checked"`
	ResponseTimeMs int       `gorm:"not null" json:"response_time_ms"`
	Details        string    `gorm:"type:text" json:"details"`
}

type APIGroup struct {
	Item []Item `json:"item"`
}

type Item struct {
	Name string `json:"name"`
}

type StatusRequest struct {
	APIGroup APIGroup `json:"collection"`
}

func (a *APIStatus) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &a)

	if err != nil {
		return err
	}

	return nil
}

func (a *APIStatus) GetAPIStatuses(db *gorm.DB, c *gin.Context) ([]APIStatus, postgresql.PaginationResponse, error) {
	var apiStatusList []APIStatus

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&apiStatusList,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return apiStatusList, paginationResponse, nil
}
