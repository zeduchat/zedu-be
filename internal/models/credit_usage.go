package models

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

var MapPackagePriceID map[string]string

func SetMapPackagePriceID(stripeConfig config.Stripe) {
	MapPackagePriceID = map[string]string{
		"starter pack":    stripeConfig.STRIPE_BASIC_CREDIT_ID,
		"pro bundle":      stripeConfig.STRIPE_ADVANCED_CREDIT_ID,
		"enterprise pack": stripeConfig.STRIPE_PREMIUM_CREDIT_ID,
	}
}

type CreditUsage struct {
	ID             string       `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	OrganisationID string       `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Amount         float64      `gorm:"type:decimal(10,2);not null" json:"amount"`
	AgentID        string       `gorm:"type:uuid;not null;index" json:"agent_id"`
	UserID         *string      `gorm:"type:uuid;index" json:"user_id"`
	CreatedAt      time.Time    `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	Organisation   Organisation `gorm:"foreignKey:OrganisationID;references:ID"`
}

type CreditUsageResponse struct {
	ID             string    `json:"id"`
	OrganisationID string    `json:"organisation_id"`
	Amount         float64   `json:"amount"`
	OrgName        string    `json:"org_name"`
	UserName       string    `json:"user_name"`
	AgentName      string    `json:"agent_name"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreditTransaction struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganisationID string    `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Amount         float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	BalanceBefore  float64   `gorm:"type:decimal(10,2);not null" json:"balance_before"`
	BalanceAfter   float64   `gorm:"type:decimal(10,2);not null" json:"balance_after"`
	Type           string    `gorm:"type:varchar(50);not null" json:"type"` // e.g., "topup", "refund"
	CreatedAt      time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type CreditPackage struct {
	ID        string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name      string    `gorm:"not null;unique" json:"name"` // e.g., "Starter Pack", "Pro Bundle"
	Credits   int       `gorm:"not null" json:"credits"`
	Price     float64   `gorm:"not null" json:"price"`
	Currency  string    `gorm:"not null;default:'USD'" json:"currency"` // e.g., USD, NGN
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type CreditPackageResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Credits  int     `json:"credits"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

type CreditTopUpRequest struct {
	OrgID     string `json:"org_id" validate:"required"`
	PackageID string `json:"package_id" validate:"required"`
	Email     string `json:"email" validate:"required"`
}

func (c *CreditTransaction) CreateCreditTransaction(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &c)
	if err != nil {
		return err
	}
	return nil
}

func (c *CreditUsage) UpdateOrCreateDailyCredit(db *gorm.DB, amount float64) error {
	var existing CreditUsage
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := db.Where("agent_id = ? AND organisation_id = ? AND created_at >= ? AND created_at < ?",
		c.AgentID, c.OrganisationID, startOfDay, endOfDay).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		c.Amount = amount
		if err := db.Create(c).Error; err != nil {
			return err
		}
	} else if err == nil {
		existing.Amount += amount
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func OrgHasValidCreditBalance(db *gorm.DB, organisationID string, creditUsed float64) bool {
	var org Organisation

	err := db.First(&org, "id = ?", organisationID).Error
	if err != nil {
		return false
	}

	if org.CreditBalance <= creditUsed {
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

	return db.Model(&Organisation{}).
		Where("id = ?", organisationID).
		Update("credit_balance", balance).Error
}

func TopUpOrgCredit(db *gorm.DB, OrgID string, PackageID string) (*gin.H, int, error) {
	var org Organisation

	org, err := org.GetOrgByID(db, OrgID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("org not found")
	}

	var credit_pkg CreditPackage

	err = db.Where("id = ?", PackageID).First(&credit_pkg).Error
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("credit package does not exist: %v", err)
	}

	// create credit transaction
	credit_transaction := CreditTransaction{
		ID:             utility.GenerateUUID(),
		OrganisationID: OrgID,
		Amount:         float64(credit_pkg.Credits),
		BalanceBefore:  float64(org.CreditBalance),
		BalanceAfter:   float64(org.CreditBalance) + float64(credit_pkg.Credits),
		Type:           "Top-up",
	}

	err = credit_transaction.CreateCreditTransaction(db)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("unable to create credit transaction: %v", err)
	}

	if err = UpdateOrgCreditBalance(db, OrgID); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("organisation credit recalculation failed: %v", err)
	}

	// Publish real-time update to superadmin dashboard (async)
	go PublishPlatformCreditUpdate(db)

	// refetch org with updated values
	org, err = org.GetOrgByID(db, OrgID)
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
	if err := db.Model(&CreditTransaction{}).
		Where("organisation_id = ?", orgID).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalCredit).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to calculate total credit: %w", err)
	}

	// Step 2: Calculate total usage
	var totalUsage float64
	if err := db.Model(&CreditUsage{}).
		Where("organisation_id = ?", orgID).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalUsage).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to calculate total usage: %w", err)
	}

	balance := totalCredit - totalUsage

	var recentTopUps []CreditTransaction
	_ = db.Where("organisation_id = ?", orgID).
		Order("created_at DESC").Limit(5).
		Find(&recentTopUps)

	var recentUsages []CreditUsage
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

func GetCreditPackages(db *gorm.DB) (*[]CreditPackageResponse, int, error) {
	var creditPackages []CreditPackage

	if err := db.Find(&creditPackages).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to get credit packages: %w", err)
	}

	var response []CreditPackageResponse
	for _, credit_packages := range creditPackages {
		response = append(response, CreditPackageResponse{
			ID:       credit_packages.ID,
			Name:     credit_packages.Name,
			Price:    credit_packages.Price,
			Credits:  credit_packages.Credits,
			Currency: credit_packages.Currency,
		})
	}

	return &response, http.StatusOK, nil
}

func GetCreditPackageByID(db *gorm.DB, id string) (*CreditPackage, int, error) {
	var creditPackage CreditPackage

	exists := postgresql.CheckExists(db, &creditPackage, "id = ?", id)
	if !exists {
		return nil, http.StatusNotFound, fmt.Errorf("credit package not found")
	}

	return &creditPackage, http.StatusOK, nil
}

func CalculateCreditCost(inputLength int, agentPrice float64) float64 {
	const (
		BaseCost     = 0.5
		InputWeight  = 0.01
		OutputWeight = 0.02
		MaxCreditCap = 50.0
	)

	// Calculate cost based on message and agent price
	rawCost := BaseCost +
		(float64(inputLength) * InputWeight) +
		(OutputWeight) +
		agentPrice

	if rawCost > MaxCreditCap {
		rawCost = MaxCreditCap
	}

	return math.Round(rawCost*100) / 100
}

func GetOrgCreditTransactions(org_id string, db *gorm.DB, c *gin.Context) ([]CreditTransaction, postgresql.PaginationResponse, error) {
	var creditTransanction []CreditTransaction

	query := db.Model(&CreditTransaction{}).
		Where("organisation_id = ?", org_id).
		Order("created_at DESC")

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&creditTransanction,
		nil,
	)
	if err != nil {
		return creditTransanction, paginationResponse, err
	}

	return creditTransanction, paginationResponse, nil
}

func GetOrgCreditUsage(orgID string, db *gorm.DB, c *gin.Context) ([]CreditUsageResponse, postgresql.PaginationResponse, error) {
	var creditUsages []CreditUsage
	var creditUsageResponses []CreditUsageResponse

	pagination := postgresql.GetPagination(c)

	query := db.Model(&CreditUsage{}).
		Where("organisation_id = ?", orgID).
		Preload("Agent").
		Order("created_at DESC")

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&creditUsages,
		nil,
	)
	if err != nil {
		return creditUsageResponses, paginationResponse, err
	}

	for _, usage := range creditUsages {

		orgAgent := OrganisationIntegrations{}
		orgAgent.CheckAgentExists(db, usage.AgentID, usage.OrganisationID)
		creditUsageResponses = append(creditUsageResponses, CreditUsageResponse{
			ID:             usage.ID,
			OrganisationID: usage.OrganisationID,
			Amount:         usage.Amount,
			AgentName:      orgAgent.AppName,
			CreatedAt:      usage.CreatedAt,
		})
	}

	return creditUsageResponses, paginationResponse, nil
}

func GetAllCreditUsage(db *gorm.DB, c *gin.Context) ([]CreditUsageResponse, postgresql.PaginationResponse, error) {
	var creditUsages []CreditUsage
	var creditUsageResponses []CreditUsageResponse

	pagination := postgresql.GetPagination(c)

	query := db.Model(&CreditUsage{}).
		Preload("Agent").
		Preload("Organisation").
		Order("created_at DESC")

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&creditUsages,
		nil,
	)
	if err != nil {
		return creditUsageResponses, paginationResponse, err
	}

	for _, usage := range creditUsages {

		orgAgent := OrganisationIntegrations{}
		orgAgent.CheckAgentExists(db, usage.AgentID, usage.OrganisationID)
		creditUsageResponses = append(creditUsageResponses, CreditUsageResponse{
			ID:             usage.ID,
			OrganisationID: usage.OrganisationID,
			Amount:         usage.Amount,
			AgentName:      orgAgent.AppName,
			OrgName:        usage.Organisation.Name,
			CreatedAt:      usage.CreatedAt,
		})
	}

	return creditUsageResponses, paginationResponse, nil
}

type PlatformCreditMetrics struct {
	TotalCredited       float64 `json:"total_credited"`
	TotalUsed           float64 `json:"total_used"`
	TotalBalance        float64 `json:"total_balance"`
	TotalOrganizations  int64   `json:"total_organizations"`
	ActiveOrganizations int64   `json:"active_organizations"`
}

func GetPlatformCreditSummary(db *gorm.DB) (PlatformCreditMetrics, error) {
	var metrics PlatformCreditMetrics

	if err := db.Model(&CreditTransaction{}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&metrics.TotalCredited).Error; err != nil {
		return metrics, fmt.Errorf("failed to calculate total credited: %w", err)
	}

	if err := db.Model(&CreditUsage{}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&metrics.TotalUsed).Error; err != nil {
		return metrics, fmt.Errorf("failed to calculate total used: %w", err)
	}

	metrics.TotalBalance = metrics.TotalCredited - metrics.TotalUsed

	if err := db.Model(&Organisation{}).
		Count(&metrics.TotalOrganizations).Error; err != nil {
		return metrics, fmt.Errorf("failed to count organizations: %w", err)
	}

	if err := db.Model(&Organisation{}).
		Where("credit_balance > ?", 0).
		Count(&metrics.ActiveOrganizations).Error; err != nil {
		return metrics, fmt.Errorf("failed to count active organizations: %w", err)
	}

	return metrics, nil
}

func PublishPlatformCreditUpdate(db *gorm.DB) {
	metrics, err := GetPlatformCreditSummary(db)
	if err != nil {
		return
	}

	channelID := "superadmin:dashboard:credits"

	if err := centrifuge.PublishChannelOptional(nil, channelID, metrics); err != nil {
		return
	}
}
