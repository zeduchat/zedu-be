package models

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
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
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganisationID  string    `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Amount          float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	BalanceBefore   float64   `gorm:"type:decimal(10,2);not null" json:"balance_before"`
	BalanceAfter    float64   `gorm:"type:decimal(10,2);not null" json:"balance_after"`
	Type            string    `gorm:"type:varchar(50);not null" json:"type"` // e.g., "topup", "refund"
	StripeSessionID *string   `gorm:"type:varchar(255);unique;index;default:null" json:"stripe_session_id"`
	CreatedAt       time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
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
	OrgID     string `json:"org_id"`
	PackageID string `json:"package_id" validate:"required"`
	Email     string `json:"email" validate:"required"`
	UserId    string `json:"-"`
}

type VerifyPaymentRequest struct {
	SessionID string `json:"session_id" validate:"required"`
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

func TopUpOrgCredit(db *gorm.DB, OrgID string, PackageID string, sessionID *string) (*gin.H, int, error) {
	var org Organisation

	org, err := org.GetOrgByID(db, OrgID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("org not found")
	}

	var credit_pkg CreditPackage

	if PackageID == "" {
		return nil, http.StatusBadRequest, errors.New("package id is required")
	}

	err = db.Where("id = ?", PackageID).First(&credit_pkg).Error
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("credit package does not exist: %v", err)
	}

	// create credit transaction
	credit_transaction := CreditTransaction{
		ID:              utility.GenerateUUID(),
		OrganisationID:  OrgID,
		Amount:          float64(credit_pkg.Credits),
		BalanceBefore:   float64(org.CreditBalance),
		BalanceAfter:    float64(org.CreditBalance) + float64(credit_pkg.Credits),
		Type:            "Top-up",
		StripeSessionID: sessionID,
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
	creditUsageResponses := make([]CreditUsageResponse, 0)

	pagination := postgresql.GetPagination(c)

	query := db.Model(&CreditUsage{}).
		Where("organisation_id = ?", orgID).
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
	creditUsageResponses := make([]CreditUsageResponse, 0)

	pagination := postgresql.GetPagination(c)

	query := db.Model(&CreditUsage{}).
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

type AICreditPackageStats struct {
	TotalCreditsSold     int64   `json:"total_credits_sold"`
	MonthlyCreditRevenue float64 `json:"monthly_credit_revenue"`
	UnusedCredits        float64 `json:"unused_credits"`
	CreditBurnRate       float64 `json:"credit_burn_rate"`
}

func GetAICreditPackageStats(db *gorm.DB) (AICreditPackageStats, error) {
	var stats AICreditPackageStats
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Total credits sold: count of top-up transactions
	if err := db.Model(&CreditTransaction{}).
		Where("type = ?", "Top-up").
		Count(&stats.TotalCreditsSold).Error; err != nil {
		return stats, fmt.Errorf("failed to count total credits sold: %w", err)
	}

	// Monthly credit revenue: sum of top-up amounts this month
	if err := db.Model(&CreditTransaction{}).
		Where("type = ? AND created_at >= ?", "Top-up", startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&stats.MonthlyCreditRevenue).Error; err != nil {
		return stats, fmt.Errorf("failed to calculate monthly credit revenue: %w", err)
	}

	// Unused credits: sum of all org credit balances
	if err := db.Model(&Organisation{}).
		Select("COALESCE(SUM(credit_balance), 0)").
		Scan(&stats.UnusedCredits).Error; err != nil {
		return stats, fmt.Errorf("failed to calculate unused credits: %w", err)
	}

	// Credit burn rate: average daily credit usage this month
	var totalUsageThisMonth float64
	if err := db.Model(&CreditUsage{}).
		Where("created_at >= ?", startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalUsageThisMonth).Error; err != nil {
		return stats, fmt.Errorf("failed to calculate credit burn rate: %w", err)
	}

	daysElapsed := now.Sub(startOfMonth).Hours() / 24
	if daysElapsed < 1 {
		daysElapsed = 1
	}
	stats.CreditBurnRate = math.Round((totalUsageThisMonth/daysElapsed)*100) / 100

	return stats, nil
}

// --- Admin Transactions History (unified view) ---

type AdminTransaction struct {
	ID            string    `json:"id"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email"`
	Type          string    `json:"type"`
	Amount        float64   `json:"amount"`
	Date          time.Time `json:"date"`
	Status        string    `json:"status"`
}

type AdminTransactionFilters struct {
	Search   string
	Duration string
	Type     string
	Status   string
	Page     int
	PageSize int
}

type AdminTransactionStatsMetric struct {
	Value            float64 `json:"value"`
	PercentageChange float64 `json:"percentage_change"`
}

type AdminTransactionStats struct {
	TotalRevenue       AdminTransactionStatsMetric `json:"total_revenue"`
	TotalTransactions  AdminTransactionStatsMetric `json:"total_transactions"`
	SuccessfulPayments AdminTransactionStatsMetric `json:"successful_payments"`
	Refunds            AdminTransactionStatsMetric `json:"refunds"`
}

func buildDurationFilter(duration string) time.Time {
	now := time.Now()
	switch duration {
	case "last_7_days":
		return now.AddDate(0, 0, -7)
	case "last_30_days":
		return now.AddDate(0, 0, -30)
	case "last_3_months":
		return now.AddDate(0, -3, 0)
	case "last_6_months":
		return now.AddDate(0, -6, 0)
	case "last_year":
		return now.AddDate(-1, 0, 0)
	default:
		return time.Time{} // no filter
	}
}

func GetAdminTransactionsHistory(db *gorm.DB, filters AdminTransactionFilters) ([]AdminTransaction, int64, error) {
	var transactions []AdminTransaction
	var totalCount int64

	// Build the UNION query for both credit pack and subscription transactions
	baseQuery := `
		SELECT id, customer_name, customer_email, type, amount, date, status FROM (
			SELECT
				ct.id as id,
				COALESCE(o.name, '') as customer_name,
				COALESCE(o.email, '') as customer_email,
				'CREDIT_PACK' as type,
				ct.amount as amount,
				ct.created_at as date,
				CASE
					WHEN ct.type = 'Refund' THEN 'Refund'
					WHEN psw.session_id IS NOT NULL THEN 'Paid'
					WHEN ct.stripe_session_id IS NULL THEN 'Paid'
					ELSE 'Pending'
				END as status
			FROM credit_transactions ct
			LEFT JOIN organisations o ON ct.organisation_id = o.id
			LEFT JOIN processed_stripe_webhooks psw ON ct.stripe_session_id = psw.session_id
			WHERE ct.updated_at IS NOT NULL

			UNION ALL

			SELECT
				op.id as id,
				COALESCE(o.name, '') as customer_name,
				COALESCE(o.email, '') as customer_email,
				'SUBSCRIPTION' as type,
				COALESCE(p.fee, 0) as amount,
				op.created_at as date,
				CASE
					WHEN op.status = 'Active' THEN 'Paid'
					WHEN op.status = 'Inactive' THEN 'Paid'
					ELSE 'Pending'
				END as status
			FROM organisation_plans op
			LEFT JOIN organisations o ON op.organisation_id = o.id
			LEFT JOIN plans p ON op.plan_id::uuid = p.id
			WHERE op.deleted_at IS NULL
		) unified`

	var whereClauses []string
	var args []interface{}

	// Duration filter
	if filters.Duration != "" {
		startDate := buildDurationFilter(filters.Duration)
		if !startDate.IsZero() {
			whereClauses = append(whereClauses, "date >= ?")
			args = append(args, startDate)
		}
	}

	// Type filter
	if filters.Type != "" {
		switch filters.Type {
		case "subscription":
			whereClauses = append(whereClauses, "type = ?")
			args = append(args, "SUBSCRIPTION")
		case "credit_pack":
			whereClauses = append(whereClauses, "type = ?")
			args = append(args, "CREDIT_PACK")
		}
	}

	// Status filter
	if filters.Status != "" {
		switch filters.Status {
		case "paid":
			whereClauses = append(whereClauses, "status = ?")
			args = append(args, "Paid")
		case "pending":
			whereClauses = append(whereClauses, "status = ?")
			args = append(args, "Pending")
		case "failed":
			whereClauses = append(whereClauses, "status = ?")
			args = append(args, "Failed")
		}
	}

	// Search filter
	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		whereClauses = append(whereClauses, "(LOWER(customer_name) LIKE LOWER(?) OR LOWER(customer_email) LIKE LOWER(?) OR LOWER(id::text) LIKE LOWER(?))")
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM (" + baseQuery + whereSQL + ") counted"
	if err := db.Raw(countQuery, args...).Scan(&totalCount).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Paginated data query
	if filters.PageSize <= 0 {
		filters.PageSize = 10
	}
	if filters.Page <= 0 {
		filters.Page = 1
	}
	offset := (filters.Page - 1) * filters.PageSize

	dataQuery := baseQuery + whereSQL + " ORDER BY date DESC LIMIT ? OFFSET ?"
	dataArgs := append(args, filters.PageSize, offset)

	if err := db.Raw(dataQuery, dataArgs...).Scan(&transactions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return transactions, totalCount, nil
}

func GetAdminTransactionStats(db *gorm.DB) (AdminTransactionStats, error) {
	var stats AdminTransactionStats
	now := time.Now()

	startOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfLastMonth := startOfCurrentMonth.AddDate(0, -1, 0)

	// --- Total Revenue ---
	var currentRevenue, lastMonthRevenue float64

	// Credit pack revenue (current month)
	db.Model(&CreditTransaction{}).
		Where("type = ? AND created_at >= ?", "Top-up", startOfCurrentMonth).
		Select("COALESCE(SUM(amount), 0)").Scan(&currentRevenue)

	// Subscription revenue (current month)
	var subRevenue float64
	db.Raw(`SELECT COALESCE(SUM(p.fee), 0) FROM organisation_plans op
		JOIN plans p ON op.plan_id::uuid = p.id
		WHERE op.created_at >= ? AND op.deleted_at IS NULL`, startOfCurrentMonth).Scan(&subRevenue)
	currentRevenue += subRevenue

	// Last month revenue
	db.Model(&CreditTransaction{}).
		Where("type = ? AND created_at >= ? AND created_at < ?", "Top-up", startOfLastMonth, startOfCurrentMonth).
		Select("COALESCE(SUM(amount), 0)").Scan(&lastMonthRevenue)

	var lastSubRevenue float64
	db.Raw(`SELECT COALESCE(SUM(p.fee), 0) FROM organisation_plans op
		JOIN plans p ON op.plan_id::uuid = p.id
		WHERE op.created_at >= ? AND op.created_at < ? AND op.deleted_at IS NULL`, startOfLastMonth, startOfCurrentMonth).Scan(&lastSubRevenue)
	lastMonthRevenue += lastSubRevenue

	stats.TotalRevenue.Value = currentRevenue
	if lastMonthRevenue > 0 {
		stats.TotalRevenue.PercentageChange = math.Round(((currentRevenue-lastMonthRevenue)/lastMonthRevenue)*1000) / 10
	} else if currentRevenue > 0 {
		stats.TotalRevenue.PercentageChange = 100
	}

	// --- Total Transactions ---
	var currentTxCount, lastMonthTxCount int64

	db.Model(&CreditTransaction{}).Where("created_at >= ?", startOfCurrentMonth).Count(&currentTxCount)
	var currentSubCount int64
	db.Model(&OrganisationPlan{}).Where("created_at >= ? AND deleted_at IS NULL", startOfCurrentMonth).Count(&currentSubCount)
	currentTxCount += currentSubCount

	db.Model(&CreditTransaction{}).Where("created_at >= ? AND created_at < ?", startOfLastMonth, startOfCurrentMonth).Count(&lastMonthTxCount)
	var lastSubCount int64
	db.Model(&OrganisationPlan{}).Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", startOfLastMonth, startOfCurrentMonth).Count(&lastSubCount)
	lastMonthTxCount += lastSubCount

	stats.TotalTransactions.Value = float64(currentTxCount)
	if lastMonthTxCount > 0 {
		stats.TotalTransactions.PercentageChange = math.Round((float64(currentTxCount-lastMonthTxCount)/float64(lastMonthTxCount))*1000) / 10
	} else if currentTxCount > 0 {
		stats.TotalTransactions.PercentageChange = 100
	}

	// --- Successful Payments ---
	// Credit transactions that have been processed (have webhook record or no stripe session = manual)
	var currentPaidCredits int64
	db.Raw(`SELECT COUNT(*) FROM credit_transactions ct
		LEFT JOIN processed_stripe_webhooks psw ON ct.stripe_session_id = psw.session_id
		WHERE ct.created_at >= ? AND ct.type != 'Refund' AND (psw.session_id IS NOT NULL OR ct.stripe_session_id IS NULL)`,
		startOfCurrentMonth).Scan(&currentPaidCredits)

	// Active subscriptions created this month
	var currentPaidSubs int64
	db.Model(&OrganisationPlan{}).
		Where("created_at >= ? AND status IN (?, ?) AND deleted_at IS NULL", startOfCurrentMonth, "Active", "Inactive").
		Count(&currentPaidSubs)

	currentSuccessful := currentPaidCredits + currentPaidSubs

	var lastPaidCredits int64
	db.Raw(`SELECT COUNT(*) FROM credit_transactions ct
		LEFT JOIN processed_stripe_webhooks psw ON ct.stripe_session_id = psw.session_id
		WHERE ct.created_at >= ? AND ct.created_at < ? AND ct.type != 'Refund' AND (psw.session_id IS NOT NULL OR ct.stripe_session_id IS NULL)`,
		startOfLastMonth, startOfCurrentMonth).Scan(&lastPaidCredits)

	var lastPaidSubs int64
	db.Model(&OrganisationPlan{}).
		Where("created_at >= ? AND created_at < ? AND status IN (?, ?) AND deleted_at IS NULL", startOfLastMonth, startOfCurrentMonth, "Active", "Inactive").
		Count(&lastPaidSubs)

	lastSuccessful := lastPaidCredits + lastPaidSubs

	stats.SuccessfulPayments.Value = float64(currentSuccessful)
	if lastSuccessful > 0 {
		stats.SuccessfulPayments.PercentageChange = math.Round((float64(currentSuccessful-lastSuccessful)/float64(lastSuccessful))*1000) / 10
	} else if currentSuccessful > 0 {
		stats.SuccessfulPayments.PercentageChange = 100
	}

	// --- Refunds ---
	var totalRefunds int64
	db.Model(&CreditTransaction{}).Where("type = ?", "Refund").Count(&totalRefunds)

	stats.Refunds.Value = float64(totalRefunds)
	// Refund percentage = refunds / total all-time transactions
	var totalAllTimeTx int64
	db.Model(&CreditTransaction{}).Count(&totalAllTimeTx)
	var totalAllTimeSubs int64
	db.Model(&OrganisationPlan{}).Where("deleted_at IS NULL").Count(&totalAllTimeSubs)
	totalAll := totalAllTimeTx + totalAllTimeSubs
	if totalAll > 0 {
		stats.Refunds.PercentageChange = math.Round((float64(totalRefunds)/float64(totalAll))*1000) / 10
	}

	return stats, nil
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
