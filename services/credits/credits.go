package credit

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

func OrgHasValidCreditBalance(db *gorm.DB, organisationID string, logger *utility.Logger) bool {
	var org models.Organisation
	var tempCredit float64 = 5

	err := db.First(&org, "id = ?", organisationID).Error
	if err != nil {
		logger.Error("Failed to get organisation")
		return false
	}

	if org.CreditBalance <= tempCredit {
		logger.Error("Organisation has insufficient credit balance!!")
		return false
	}

	return true
}

func CalculateOrgCreditBalance(db *gorm.DB, organisationID string) (float64, error) {
	var balance float64

	query := `
	SELECT 
		(SELECT COALESCE(SUM(amount), 0) FROM credit_transactions WHERE organisation_id = ?) -
		(SELECT COALESCE(SUM(amount), 0) FROM credit_usages WHERE organisation_id = ?) 
		AS balance
	`

	err := db.Raw(query, organisationID, organisationID).Scan(&balance).Error
	if err != nil {
		return 0, fmt.Errorf("failed to fetch credit balance: %w", err)
	}

	return balance, nil
}

func UpdateOrgCreditBalance(db *gorm.DB, organisationID string) error {
	balance, err := CalculateOrgCreditBalance(db, organisationID)
	if err != nil {
		return err
	}

	return db.Model(&models.Organisation{}).
		Where("id = ?", organisationID).
		Update("credit_balance", balance).Error
}

func TopUpOrgCredit(req models.CreditTopUpRequest, db *gorm.DB) (*gin.H, int, error) {
	var org models.Organisation

	org, err := org.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("org not found")
	}

	var credit_pkg models.CreditPackage

	err = db.Where("id = ?", req.PackageID).First(&credit_pkg).Error
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("credit package does not exist: %v", err)
	}

	// create credit transaction
	credit_transaction := models.CreditTransaction{
		ID:             utility.GenerateUUID(),
		OrganisationID: req.OrgID,
		Amount:         float64(credit_pkg.Credits),
		BalanceBefore:  float64(org.CreditBalance),
		BalanceAfter:   float64(org.CreditBalance) + float64(credit_pkg.Credits),
		Type:           "Top-up",
	}

	err = credit_transaction.CreateCreditTransaction(db)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("unable to create credit transaction: %v", err)
	}

	if err = UpdateOrgCreditBalance(db, req.OrgID); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("organisation credit recalculation failed: %v", err)
	}

	// refetch org with updated values
	org, err = org.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to fetch updated organisation details")
	}

	responseData := gin.H{
		"organisation": org,
	}

	return &responseData, http.StatusOK, nil
}

func GetOrgCreditReport(orgID string, db *gorm.DB) (*gin.H, int, error) {

	// Step 1: Calculate total top-ups
	var totalCredit float64
	if err := db.Model(&models.CreditTransaction{}).
		Where("organisation_id = ?", orgID).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalCredit).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to calculate total credit: %w", err)
	}

	// Step 2: Calculate total usage
	var totalUsage float64
	if err := db.Model(&models.CreditUsage{}).
		Where("organisation_id = ?", orgID).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalUsage).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to calculate total usage: %w", err)
	}

	balance := totalCredit - totalUsage

	var recentTopUps []models.CreditTransaction
	_ = db.Where("organisation_id = ?", orgID).
		Order("created_at DESC").Limit(5).
		Find(&recentTopUps)

	var recentUsages []models.CreditUsage
	_ = db.Where("organisation_id = ?", orgID).
		Order("created_at DESC").Limit(5).
		Find(&recentUsages)

	response := gin.H{
		"organisation_id": orgID,
		"total_credited":  totalCredit,
		"total_used":      totalUsage,
		"balance":         balance,
		"recent_topups":   recentTopUps,
		"recent_usages":   recentUsages,
	}

	return &response, http.StatusOK, nil
}
