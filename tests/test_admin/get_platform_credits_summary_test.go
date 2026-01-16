package test_admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func TestGetPlatformCreditsSummary_Success(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	r, _, _, _ := SetupAdminTestRouter()
	token := CreateSuperAdminAndGetToken(t, r, db)

	org1ID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	org2ID := CreateOrganizationWithCredit(t, db.Postgresql, 200.00)
	org3ID := CreateOrganizationWithCredit(t, db.Postgresql, 300.00)
	_ = CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	org5ID := CreateOrganizationWithCredit(t, db.Postgresql, 500.00)

	CreateCreditTransaction(t, db.Postgresql, org1ID, 100.00)
	CreateCreditTransaction(t, db.Postgresql, org2ID, 200.00)
	CreateCreditTransaction(t, db.Postgresql, org3ID, 300.00)
	CreateCreditTransaction(t, db.Postgresql, org5ID, 500.00)

	CreateCreditUsage(t, db.Postgresql, org1ID, 20.00)
	CreateCreditUsage(t, db.Postgresql, org2ID, 50.00)
	CreateCreditUsage(t, db.Postgresql, org3ID, 100.00)
	CreateCreditUsage(t, db.Postgresql, org5ID, 150.00)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])
	assert.NotNil(t, data["data"])

	metrics := data["data"].(map[string]any)

	_, hasTotalCredited := metrics["total_credited"]
	_, hasTotalUsed := metrics["total_used"]
	_, hasTotalBalance := metrics["total_balance"]
	_, hasTotalOrganizations := metrics["total_organizations"]
	_, hasActiveOrganizations := metrics["active_organizations"]

	assert.True(t, hasTotalCredited, "total_credited field missing")
	assert.True(t, hasTotalUsed, "total_used field missing")
	assert.True(t, hasTotalBalance, "total_balance field missing")
	assert.True(t, hasTotalOrganizations, "total_organizations field missing")
	assert.True(t, hasActiveOrganizations, "active_organizations field missing")

	totalCredited := metrics["total_credited"].(float64)
	totalUsed := metrics["total_used"].(float64)
	totalBalance := metrics["total_balance"].(float64)

	assert.Greater(t, totalCredited, float64(1000.00))
	assert.Greater(t, totalUsed, float64(300.00))
	assert.Greater(t, totalBalance, float64(500.00))
	assert.GreaterOrEqual(t, int64(metrics["total_organizations"].(float64)), int64(5))
	assert.GreaterOrEqual(t, int64(metrics["active_organizations"].(float64)), int64(4))
}

func TestGetPlatformCreditsSummary_Unauthorized_NoToken(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	r, _, _, _ := SetupAdminTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	data := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", data["status"])
}

func TestGetPlatformCreditsSummary_Unauthorized_InvalidToken(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	r, _, _, _ := SetupAdminTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer invalid_token_here")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	data := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", data["status"])
}

func TestGetPlatformCreditsSummary_ZeroCredits(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	r, _, _, _ := SetupAdminTestRouter()
	token := CreateSuperAdminAndGetToken(t, r, db)

	_ = CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	_ = CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	_ = CreateOrganizationWithCredit(t, db.Postgresql, 0.00)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	data := tst.ParseResponse(rr)
	metrics := data["data"].(map[string]any)

	totalUsed := metrics["total_used"].(float64)
	totalCredited := metrics["total_credited"].(float64)
	totalBalance := metrics["total_balance"].(float64)

	assert.Equal(t, float64(0.00), totalUsed)
	assert.GreaterOrEqual(t, totalBalance, float64(-3.00))

	orgCount := int64(metrics["total_organizations"].(float64))
	assert.GreaterOrEqual(t, orgCount, int64(3))

	assert.GreaterOrEqual(t, totalCredited, float64(0.00))
}

func TestGetPlatformCreditsSummary_MultipleOrganizations(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	r, _, _, _ := SetupAdminTestRouter()
	token := CreateSuperAdminAndGetToken(t, r, db)

	for i := 0; i < 20; i++ {
		orgID := CreateOrganizationWithCredit(t, db.Postgresql, float64(i*10))
		CreateCreditTransaction(t, db.Postgresql, orgID, float64(i*10))
		CreateCreditUsage(t, db.Postgresql, orgID, float64(i*2))
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	data := tst.ParseResponse(rr)
	metrics := data["data"].(map[string]any)

	totalCredited := float64(0)
	for i := 0; i < 20; i++ {
		totalCredited += float64(i * 10)
	}

	totalUsed := float64(0)
	for i := 0; i < 20; i++ {
		totalUsed += float64(i * 2)
	}

	assert.GreaterOrEqual(t, metrics["total_credited"].(float64), totalCredited)
	assert.GreaterOrEqual(t, metrics["total_used"].(float64), totalUsed)
	assert.GreaterOrEqual(t, metrics["total_balance"].(float64), totalCredited-totalUsed)
	assert.GreaterOrEqual(t, int64(metrics["total_organizations"].(float64)), int64(20))
}

func TestGetPlatformCreditsSummary_ActiveOrganizationsCount(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	r, _, _, _ := SetupAdminTestRouter()
	token := CreateSuperAdminAndGetToken(t, r, db)

	org1ID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	org2ID := CreateOrganizationWithCredit(t, db.Postgresql, 200.00)
	org3ID := CreateOrganizationWithCredit(t, db.Postgresql, 300.00)

	CreateCreditTransaction(t, db.Postgresql, org1ID, 100.00)
	CreateCreditTransaction(t, db.Postgresql, org2ID, 200.00)
	CreateCreditTransaction(t, db.Postgresql, org3ID, 300.00)

	CreateCreditUsage(t, db.Postgresql, org1ID, 50.00)

	_ = CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	_ = CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	_ = CreateOrganizationWithCredit(t, db.Postgresql, -10.00)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	data := tst.ParseResponse(rr)
	metrics := data["data"].(map[string]any)

	assert.GreaterOrEqual(t, int64(metrics["total_organizations"].(float64)), int64(6))
	assert.GreaterOrEqual(t, int64(metrics["active_organizations"].(float64)), int64(2))
}
