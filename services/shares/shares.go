package shares

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateShares(db *gorm.DB, req models.ShareRequest, userID string) (*models.Shares, error) {
	share := &models.Shares{
		ID:             utility.GenerateUUID(),
		UserID:         userID,
		NumberOfShares: req.NumberOfShares,
		PricePurchased: req.PricePurchased,
	}
	err := share.Create(db)
	if err != nil {
		return nil, err
	}
	return share, nil
}

func GetUserShares(db *gorm.DB, c *gin.Context, userID string) ([]models.Shares, postgresql.PaginationResponse, error) {
	var share models.Shares
	shares, paginationResponse, err := share.GetUserShares(db, c, userID)
	if err != nil {
		return nil, paginationResponse, err
	}
	return shares, paginationResponse, nil
}

func GetShareByID(db *gorm.DB, id string) (models.Shares, error) {
	var share models.Shares
	return share.GetByID(db, id)
}

func DeleteShares(db *gorm.DB, id string) error {
	var share models.Shares
	return share.DeleteShares(db, id)
}

// GetSharePerformance calculates the user's portfolio performance
func GetSharePerformance(db *gorm.DB, c *gin.Context, userID string) (*models.SharePortfolioResponse, error) {
	var share models.Shares
	var settings models.ShareSettings
	shares, _, err := share.GetUserShares(db, c, userID)
	if err != nil {
		return nil, err
	}

	// Get current share price
	currentPrice, err := settings.GetCurrentPrice(db)
	if err != nil {
		return nil, err
	}

	// Calculate totals
	var totalShares int
	var totalInvested float64

	for _, s := range shares {
		totalShares += s.NumberOfShares
		totalInvested += float64(s.NumberOfShares) * s.PricePurchased
	}

	// Calculate performance
	currentValue := float64(totalShares) * currentPrice
	profitLoss := currentValue - totalInvested

	var performancePercent float64
	if totalInvested > 0 {
		performancePercent = (profitLoss / totalInvested) * 100
	}

	return &models.SharePortfolioResponse{
		TotalShares:        totalShares,
		TotalInvested:      totalInvested,
		CurrentPrice:       currentPrice,
		CurrentValue:       currentValue,
		ProfitLoss:         profitLoss,
		PerformancePercent: performancePercent,
	}, nil
}
