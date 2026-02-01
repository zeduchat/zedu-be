package test_admin

import (
	"fmt"
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
	"github.com/hngprojects/telex_be/utility"
)

func SetupAdminTestRouter() (*gin.Engine, *auth.Controller, *utility.Logger, *storage.Database) {
	gin.SetMode(gin.TestMode)
	logger := utility.NewLogger()
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

func CreateSuperAdminAndGetToken(t *testing.T, r *gin.Engine, db *storage.Database) string {
	_, token := CreateSuperAdminAndGetTokenWithID(t, r, db)
	return token
}

// CreateSuperAdminAndGetTokenWithID creates a super admin and returns both the admin ID and token
// for test-specific cleanup
func CreateSuperAdminAndGetTokenWithID(t *testing.T, r *gin.Engine, db *storage.Database) (string, string) {
	password, _ := utility.HashPassword("password")

	superAdmin := models.Admin{
		ID:       utility.GenerateUUID(),
		Name:     "Super Admin Test",
		Email:    fmt.Sprintf("superadmin%s@qa.team", utility.RandomString(5)),
		Role:     models.RoleSuperAdmin,
		IsActive: true,
		Password: password,
	}

	err := db.Postgresql.Create(&superAdmin).Error
	if err != nil {
		t.Fatalf("Failed to create superadmin: %v", err)
	}

	c, _ := gin.CreateTestContext(nil)
	tokenData, err := middleware.CreateAdminToken(superAdmin, c)
	if err != nil {
		t.Fatalf("Failed to create admin token: %v", err)
	}

	accessToken := models.AccessToken{
		ID:               tokenData.AccessUuid,
		OwnerID:          superAdmin.ID,
		LoginAccessToken: tokenData.AccessToken,
		IsLive:           true,
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          "",
	}

	tokenSaveErr := accessToken.CreateAccessToken(db.Postgresql, tokens)
	if tokenSaveErr != nil {
		t.Fatalf("Failed to save access token: %v", tokenSaveErr)
	}

	return superAdmin.ID, tokenData.AccessToken
}

func CreateAdminAndGetToken(t *testing.T, r *gin.Engine, db *storage.Database, role string) string {
	password, _ := utility.HashPassword("password")

	admin := models.Admin{
		ID:       utility.GenerateUUID(),
		Name:     "Admin Test",
		Email:    fmt.Sprintf("admin%s@qa.team", utility.RandomString(5)),
		Role:     role,
		IsActive: true,
		Password: password,
	}

	err := db.Postgresql.Create(&admin).Error
	if err != nil {
		t.Fatalf("Failed to create admin: %v", err)
	}

	c, _ := gin.CreateTestContext(nil)
	tokenData, err := middleware.CreateAdminToken(admin, c)
	if err != nil {
		t.Fatalf("Failed to create admin token: %v", err)
	}

	accessToken := models.AccessToken{
		ID:               tokenData.AccessUuid,
		OwnerID:          admin.ID,
		LoginAccessToken: tokenData.AccessToken,
		IsLive:           true,
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          "",
	}

	tokenSaveErr := accessToken.CreateAccessToken(db.Postgresql, tokens)
	if tokenSaveErr != nil {
		t.Fatalf("Failed to save access token: %v", tokenSaveErr)
	}

	return tokenData.AccessToken
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

	err := db.Create(&transaction).Error
	if err != nil {
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

	err := db.Create(&usage).Error
	if err != nil {
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

	err := db.Create(&org).Error
	if err != nil {
		t.Fatalf("Failed to create organization %s", err)
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
	err := db.Create(&pkg).Error
	if err != nil {
		t.Fatalf("Failed to create credit package: %v", err)
	}
	return pkg.ID
}

func cleanupAdmins(db *gorm.DB) {
	// Get all test admin IDs
	var adminIDs []string
	db.Raw("SELECT id FROM admins WHERE email LIKE ?", "%@qa.team%").Scan(&adminIDs)

	// Delete access tokens for each admin
	for _, adminID := range adminIDs {
		db.Exec("DELETE FROM access_tokens WHERE owner_id = ?", adminID)
	}

	// Delete admins
	db.Exec("DELETE FROM admins WHERE email LIKE ?", "%@qa.team%")
}

func CleanupSpecificTestData(db *gorm.DB, adminID string, orgIDs []string) {
	// Delete child records for each org
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

	// Delete access tokens for this specific admin
	if adminID != "" {
		db.Exec("DELETE FROM access_tokens WHERE owner_id = ?", adminID)
		db.Exec("DELETE FROM admins WHERE id = ?", adminID)
	}
}
