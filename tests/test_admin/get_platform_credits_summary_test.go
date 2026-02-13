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

	// Create admin with unique email for this test
	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	// Track all created org IDs for cleanup
	var createdOrgIDs []string

	org1ID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	createdOrgIDs = append(createdOrgIDs, org1ID)
	org2ID := CreateOrganizationWithCredit(t, db.Postgresql, 200.00)
	createdOrgIDs = append(createdOrgIDs, org2ID)
	org3ID := CreateOrganizationWithCredit(t, db.Postgresql, 300.00)
	createdOrgIDs = append(createdOrgIDs, org3ID)
	org4ID := CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	createdOrgIDs = append(createdOrgIDs, org4ID)
	org5ID := CreateOrganizationWithCredit(t, db.Postgresql, 500.00)
	createdOrgIDs = append(createdOrgIDs, org5ID)

	CreateCreditTransaction(t, db.Postgresql, org1ID, 100.00)
	CreateCreditTransaction(t, db.Postgresql, org2ID, 200.00)
	CreateCreditTransaction(t, db.Postgresql, org3ID, 300.00)
	CreateCreditTransaction(t, db.Postgresql, org5ID, 500.00)

	CreateCreditUsage(t, db.Postgresql, org1ID, 20.00)
	CreateCreditUsage(t, db.Postgresql, org2ID, 50.00)
	CreateCreditUsage(t, db.Postgresql, org3ID, 100.00)
	CreateCreditUsage(t, db.Postgresql, org5ID, 150.00)

	// Setup cleanup to run AFTER test, cleaning only OUR data
	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Log response body for debugging if test fails
	if rr.Code != http.StatusOK {
		t.Logf("Response body: %s", rr.Body.String())
	}

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	if !assert.Equal(t, "success", data["status"]) {
		t.FailNow()
	}
	if !assert.NotNil(t, data["data"]) {
		t.FailNow()
	}

	metrics, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("data['data'] is not a map, got: %T", data["data"])
	}

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
	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	var createdOrgIDs []string
	org1 := CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	createdOrgIDs = append(createdOrgIDs, org1)
	org2 := CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	createdOrgIDs = append(createdOrgIDs, org2)
	org3 := CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	createdOrgIDs = append(createdOrgIDs, org3)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	metrics, ok := data["data"].(map[string]any)
	if !ok {
		t.FailNow()
	}

	totalUsed := metrics["total_used"].(float64)
	totalCredited := metrics["total_credited"].(float64)
	totalBalance := metrics["total_balance"].(float64)

	assert.GreaterOrEqual(t, totalUsed, float64(0.00))
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
	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	var createdOrgIDs []string
	for i := 0; i < 20; i++ {
		orgID := CreateOrganizationWithCredit(t, db.Postgresql, float64(i*10))
		createdOrgIDs = append(createdOrgIDs, orgID)
		CreateCreditTransaction(t, db.Postgresql, orgID, float64(i*10))
		CreateCreditUsage(t, db.Postgresql, orgID, float64(i*2))
	}

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	metrics, ok := data["data"].(map[string]any)
	if !ok {
		t.FailNow()
	}

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
	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	var createdOrgIDs []string
	org1ID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	createdOrgIDs = append(createdOrgIDs, org1ID)
	org2ID := CreateOrganizationWithCredit(t, db.Postgresql, 200.00)
	createdOrgIDs = append(createdOrgIDs, org2ID)
	org3ID := CreateOrganizationWithCredit(t, db.Postgresql, 300.00)
	createdOrgIDs = append(createdOrgIDs, org3ID)

	CreateCreditTransaction(t, db.Postgresql, org1ID, 100.00)
	CreateCreditTransaction(t, db.Postgresql, org2ID, 200.00)
	CreateCreditTransaction(t, db.Postgresql, org3ID, 300.00)

	CreateCreditUsage(t, db.Postgresql, org1ID, 50.00)

	org4ID := CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	createdOrgIDs = append(createdOrgIDs, org4ID)
	org5ID := CreateOrganizationWithCredit(t, db.Postgresql, 0.00)
	createdOrgIDs = append(createdOrgIDs, org5ID)
	org6ID := CreateOrganizationWithCredit(t, db.Postgresql, -10.00)
	createdOrgIDs = append(createdOrgIDs, org6ID)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/credits-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	metrics, ok := data["data"].(map[string]any)
	if !ok {
		t.FailNow()
	}

	assert.GreaterOrEqual(t, int64(metrics["total_organizations"].(float64)), int64(6))
	assert.GreaterOrEqual(t, int64(metrics["active_organizations"].(float64)), int64(2))
}
