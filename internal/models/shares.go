package models

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Shares struct {
	ID             string         `gorm:"type:uuid;primary_key" json:"id"`
	UserID         string         `gorm:"type:uuid;not null" json:"user_id"`
	NumberOfShares int            `gorm:"type:int;not null" json:"number_of_shares"`
	PricePurchased float64        `gorm:"type:decimal(16,2);not null" json:"price_purchased"`
	CreatedAt      time.Time      `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
type ShareRequest struct {
	NumberOfShares int     `json:"number_of_shares" validate:"required,min=1"`
	PricePurchased float64 `json:"price_purchased" validate:"required,gt=0"`
}

func (s *Shares) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Shares) GetUserShares(db *gorm.DB, c *gin.Context, userID string) ([]Shares, postgresql.PaginationResponse, error) {
	var shares []Shares
	pagination := postgresql.GetPagination(c)
	query := db.Where("user_id = ?", userID)
	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&shares,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}
	return shares, paginationResponse, nil
}

func (s *Shares) DeleteShares(db *gorm.DB, id string) error {
	err := db.Where("id = ?", id).First(&s).Error
	if err != nil {
		return err
	}
	err = postgresql.DeleteRecordFromDb(db, &s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Shares) GetByID(db *gorm.DB, id string) (Shares, error) {
	var share Shares
	err, _ := postgresql.SelectOneFromDb(db, &share, "id = ?", id)
	if err != nil {
		return share, err
	}
	return share, nil
}

// ShareSettings stores the current share price (single row table)
type ShareSettings struct {
	ID                   string    `gorm:"type:uuid;primary_key" json:"id"`
	CurrentPricePerShare float64   `gorm:"type:decimal(16,2);not null" json:"current_price_per_share"`
	CreatedAt            time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

func (ss *ShareSettings) GetCurrentPrice(db *gorm.DB) (float64, error) {
	var settings ShareSettings
	err := db.First(&settings).Error
	if err != nil {
		return 0, err
	}
	return settings.CurrentPricePerShare, nil
}

func (ss *ShareSettings) UpdateCurrentPrice(db *gorm.DB, newPrice float64) error {
	ss.CurrentPricePerShare = newPrice
	return db.Save(ss).Error
}

// SharePortfolioResponse is the response for user portfolio with calculations
type SharePortfolioResponse struct {
	TotalShares       int     `json:"total_shares"`
	TotalInvested     float64 `json:"total_invested"`
	CurrentPrice      float64 `json:"current_price_per_share"`
	CurrentValue      float64 `json:"current_value"`
	ProfitLoss        float64 `json:"profit_loss"`
	PerformancePercent float64 `json:"performance_percent"`
}
