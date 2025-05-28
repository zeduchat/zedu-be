package credit

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
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
	fmt.Println("balance")
	fmt.Println(balance)
	fmt.Println(err)
	if err != nil {
		return err
	}

	return db.Model(&models.Organisation{}).
		Where("id = ?", organisationID).
		Update("credit_balance", balance).Error
}
