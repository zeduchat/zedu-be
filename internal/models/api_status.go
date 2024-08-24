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
	Status         string    `gorm:"type:text;null" json:"status"`
	LastChecked    time.Time `gorm:"null" json:"last_checked"`
	ResponseTimeMs string    `gorm:"null" json:"response_time_ms"`
	Details        string    `gorm:"type:text" json:"details"`
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

type StatusRequest struct {
	APIGroup APIGroup `json:"collection"`
}

type APIGroup struct {
	Item []Item `json:"item"`
}

type Item struct {
	Name string    `json:"name"`
	Item []SubItem `json:"item"`
}

type SubItem struct {
	Name     string     `json:"name"`
	Response []Response `json:"response"`
}

type Response struct {
	Name       string `json:"name"`
	StatusCode int    `json:"code"`
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
		"",
		"asc",
		pagination,
		&apiStatusList,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return apiStatusList, paginationResponse, nil
}
