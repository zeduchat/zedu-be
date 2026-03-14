package test_admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/admin"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

var (
	testLogger *utility.Logger
	loggerOnce sync.Once
)

func getTestLogger() *utility.Logger {
	loggerOnce.Do(func() {
		testLogger = tst.Setup()
	})
	return testLogger
}

func SetupAdminTestRouter() (*gin.Engine, *auth.Controller, *utility.Logger, *storage.Database) {
	gin.SetMode(gin.TestMode)
	logger := getTestLogger()
	db := storage.Connection()
	validator := validator.New()

	authController := &auth.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	adminController := &admin.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	SetupAdminRoutes(r, adminController)
	return r, authController, logger, db
}

func SetupAdminRoutes(r *gin.Engine, adminController *admin.Controller) {
	apiVersion := "/api/v1"

	adminUrl := r.Group(fmt.Sprintf("%s/backoffice", apiVersion),
		middleware.AdminAuthorize(adminController.Db.Postgresql))
	{
		adminUrl.GET("/dashboard/credits-summary", adminController.GetPlatformCreditsSummary)
		adminUrl.POST("/notifications/broadcast", adminController.BroadcastNotification)

		// Billing - plans
		adminUrl.GET("/billing/stats", adminController.GetSubscriptionBillingStats)
		adminUrl.GET("/billing/plans", adminController.GetPlansFiltered)
		adminUrl.POST("/billing/plans", adminController.CreatePlan)
		adminUrl.PUT("/billing/plans/:plan_id", adminController.UpdatePlan)
		adminUrl.DELETE("/billing/plans/:plan_id", adminController.DeletePlan)

		// Billing - credit packages
		adminUrl.GET("/billing/credit-packages/stats", adminController.GetAICreditPackageStats)
		adminUrl.GET("/billing/credit-packages", adminController.GetAICreditPackagesFiltered)
		adminUrl.POST("/billing/credit-packages", adminController.CreateAICreditPackage)
		adminUrl.PUT("/billing/credit-packages/:package_id", adminController.UpdateAICreditPackage)
		adminUrl.DELETE("/billing/credit-packages/:package_id", adminController.DeleteAICreditPackage)

		// Billing - transactions
		adminUrl.GET("/billing/transactions/stats", adminController.GetAdminTransactionStats)
		adminUrl.GET("/billing/transactions", adminController.GetAdminTransactionsHistory)
	}

	superAdminAuthUrl := r.Group(fmt.Sprintf("%s/backoffice", apiVersion),
		middleware.AdminAuthorize(adminController.Db.Postgresql),
		middleware.RequireSuperAdmin())
	{
		superAdminAuthUrl.POST("/admins/:admin_id/role/initiate", adminController.InitiateChangeAdminRole)
		superAdminAuthUrl.POST("/admins/role/confirm", adminController.ConfirmChangeAdminRole)
		superAdminAuthUrl.GET("/admins/audit-logs", adminController.GetRoleAuditHistory)
	}

	loginUrl := r.Group(fmt.Sprintf("%s/backoffice", apiVersion))
	{
		loginUrl.POST("/login", adminController.LoginAdmin)
	}
}

func loginAdmin(t *testing.T, r *gin.Engine, email, password string) string {
	t.Helper()
	loginPayload := map[string]string{"email": email, "password": password}
	body, _ := json.Marshal(loginPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Admin login failed, status: %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("Login response missing 'data' field: %v", resp)
	}
	token, ok := data["access_token"].(string)
	if !ok || token == "" {
		t.Fatalf("Login response missing 'access_token': %v", data)
	}
	return token
}

func CreateSuperAdminAndGetToken(t *testing.T, r *gin.Engine, db *storage.Database) string {
	_, token := CreateSuperAdminAndGetTokenWithID(t, r, db)
	return token
}

func CreateSuperAdminAndGetTokenWithID(t *testing.T, r *gin.Engine, db *storage.Database) (string, string) {
	t.Helper()
	const rawPassword = "password"
	hashedPassword, _ := utility.HashPassword(rawPassword)

	superAdmin := models.Admin{
		ID:       utility.GenerateUUID(),
		Name:     "Super Admin Test",
		Email:    fmt.Sprintf("superadmin%s@qa.team", utility.RandomString(5)),
		Role:     models.RoleSuperAdmin,
		IsActive: true,
		Password: hashedPassword,
	}

	if err := db.Postgresql.Create(&superAdmin).Error; err != nil {
		t.Fatalf("Failed to create superadmin: %v", err)
	}

	token := loginAdmin(t, r, superAdmin.Email, rawPassword)
	return superAdmin.ID, token
}

func CreateAdminAndGetToken(t *testing.T, r *gin.Engine, db *storage.Database, role string) string {
	t.Helper()
	const rawPassword = "password"
	hashedPassword, _ := utility.HashPassword(rawPassword)

	adminUser := models.Admin{
		ID:       utility.GenerateUUID(),
		Name:     "Admin Test",
		Email:    fmt.Sprintf("admin%s@qa.team", utility.RandomString(5)),
		Role:     role,
		IsActive: true,
		Password: hashedPassword,
	}

	if err := db.Postgresql.Create(&adminUser).Error; err != nil {
		t.Fatalf("Failed to create admin: %v", err)
	}

	return loginAdmin(t, r, adminUser.Email, rawPassword)
}

func CreateCreditTransaction(t *testing.T, db *gorm.DB, orgID string, amount float64) {
	transaction := models.CreditTransaction{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         amount,
		BalanceBefore:  0,
		BalanceAfter:   amount,
		Type:           "Top-up",
	}
	if err := db.Create(&transaction).Error; err != nil {
		t.Fatalf("Failed to create credit transaction: %v", err)
	}
}

func CreateCreditUsage(t *testing.T, db *gorm.DB, orgID string, amount float64) {
	usage := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         amount,
		AgentID:        utility.GenerateUUID(),
		UserID:         nil,
	}
	if err := db.Create(&usage).Error; err != nil {
		t.Fatalf("Failed to create credit usage: %v", err)
	}
}

func CreateOrganizationWithCredit(t *testing.T, db *gorm.DB, creditBalance float64) string {
	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:            orgID,
		Name:          fmt.Sprintf("Test Org %s", utility.RandomString(5)),
		Email:         fmt.Sprintf("testorg%s@qa.team", utility.RandomString(5)),
		CreditBalance: creditBalance,
		OwnerID:       utility.GenerateUUID(),
	}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}
	if creditBalance > 0 {
		tx := models.CreditTransaction{
			ID:             utility.GenerateUUID(),
			OrganisationID: orgID,
			Amount:         creditBalance,
			BalanceBefore:  0,
			BalanceAfter:   creditBalance,
			Type:           "Top-up",
		}
		if err := db.Create(&tx).Error; err != nil {
			t.Fatalf("Failed to create initial credit transaction: %v", err)
		}
	}
	return orgID
}

func CreateTestCreditPackage(t *testing.T, db *gorm.DB, credits int, price float64) string {
	pkg := models.CreditPackage{
		ID:       utility.GenerateUUID(),
		Name:     fmt.Sprintf("Test Package %s", utility.RandomString(5)),
		Credits:  credits,
		Price:    price,
		Currency: "USD",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("Failed to create credit package: %v", err)
	}
	return pkg.ID
}

func CleanupSpecificTestData(db *gorm.DB, adminID string, orgIDs []string) {
	childTables := []struct {
		table  string
		column string
	}{
		{"invitations", "organisation_id"},
		{"credit_usages", "organisation_id"},
		{"credit_transactions", "organisation_id"},
		{"integration_bills", "org_id"},
		{"organisation_integrations", "org_id"},
		{"channels", "organisation_id"},
		{"dm_channels", "org_id"},
		{"channel_participants", "org_id"},
		{"dm_favourites", "org_id"},
		{"org_user_managements", "organisation_id"},
		{"user_organisations", "organisation_id"},
		{"organisation_plans", "organisation_id"},
		{"org_roles", "organisation_id"},
	}

	for _, orgID := range orgIDs {
		for _, child := range childTables {
			db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", child.table, child.column), orgID)
		}
		// Delete the organisation
		db.Exec("DELETE FROM organisations WHERE id = ?", orgID)
	}
	if adminID != "" {
		db.Exec("DELETE FROM access_tokens WHERE owner_id = ?", adminID)
		db.Exec("DELETE FROM admins WHERE id = ?", adminID)
	}
}
