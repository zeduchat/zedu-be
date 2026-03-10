package test_admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/admin"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	tst "github.com/hngprojects/telex_be/tests"
)

func setupBillingTestRouter() (*gin.Engine, *storage.Database) {
	gin.SetMode(gin.TestMode)
	logger := utility.NewLogger()
	db := storage.Connection()
	validatorRef := validator.New()

	adminCtl := &admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	adminUrl := r.Group("/api/v1/backoffice", middleware.AdminAuthorize(db.Postgresql))
	{
		adminUrl.GET("/billing/stats", adminCtl.GetSubscriptionBillingStats)
		adminUrl.GET("/billing/plans", adminCtl.GetPlansFiltered)
		adminUrl.POST("/billing/plans", adminCtl.CreatePlan)
		adminUrl.PUT("/billing/plans/:plan_id", adminCtl.UpdatePlan)
		adminUrl.DELETE("/billing/plans/:plan_id", adminCtl.DeletePlan)
		adminUrl.POST("/billing/credit-packages", adminCtl.CreateAICreditPackage)
		adminUrl.PUT("/billing/credit-packages/:package_id", adminCtl.UpdateAICreditPackage)
		adminUrl.DELETE("/billing/credit-packages/:package_id", adminCtl.DeleteAICreditPackage)
		adminUrl.GET("/billing/transactions", adminCtl.GetAllCreditTransactions)
	}

	return r, db
}

// --- Subscription Billing Stats ---

func TestGetSubscriptionBillingStats_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)
	var createdOrgIDs []string

	// Create orgs with active subscriptions
	for i := 0; i < 3; i++ {
		orgID := CreateOrganizationWithCredit(t, db.Postgresql, float64(i*100))
		createdOrgIDs = append(createdOrgIDs, orgID)

		// Create an active organisation plan
		var freePlan models.Plan
		db.Postgresql.Where("name = ?", "Free").First(&freePlan)
		if freePlan.ID != "" {
			orgPlan := models.OrganisationPlan{
				ID:             utility.GenerateUUID(),
				OrganisationID: orgID,
				PlanID:         freePlan.ID,
				Status:         "Active",
			}
			db.Postgresql.Create(&orgPlan)
		}
	}

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/billing/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])

	stats, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("data['data'] is not a map, got: %T", data["data"])
	}

	_, hasActiveSubs := stats["total_active_subscriptions"]
	_, hasMRR := stats["monthly_recurring_revenue"]
	_, hasDisabledSubs := stats["disabled_subscriptions"]
	_, hasChurnRate := stats["churn_rate"]

	assert.True(t, hasActiveSubs, "total_active_subscriptions field missing")
	assert.True(t, hasMRR, "monthly_recurring_revenue field missing")
	assert.True(t, hasDisabledSubs, "disabled_subscriptions field missing")
	assert.True(t, hasChurnRate, "churn_rate field missing")

	assert.GreaterOrEqual(t, stats["total_active_subscriptions"].(float64), float64(0))
	assert.GreaterOrEqual(t, stats["monthly_recurring_revenue"].(float64), float64(0))
	assert.GreaterOrEqual(t, stats["churn_rate"].(float64), float64(0))
}

func TestGetSubscriptionBillingStats_Unauthorized(t *testing.T) {
	_ = tst.Setup()
	r, _ := setupBillingTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/billing/stats", nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// --- Plan CRUD ---

func TestCreatePlan_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		// Clean up the created plan
		db.Postgresql.Exec("DELETE FROM plans WHERE name = ?", "Test Plan Create")
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	payload := models.CreatePlanRequest{
		Name:                "Test Plan Create",
		Description:         "A test plan for unit testing",
		Fee:                 50,
		Benefits:            []string{"Benefit 1", "Benefit 2"},
		MaxChannels:         10,
		MaxUsers:            100,
		MaxBuzzParticipants: 25,
		MaxActiveCalls:      10,
		AICreditsPurchasable: true,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/billing/plans", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusCreated, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])

	plan, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("data['data'] is not a map, got: %T", data["data"])
	}

	assert.Equal(t, "Test Plan Create", plan["name"])
	assert.Equal(t, "A test plan for unit testing", plan["description"])
	assert.Equal(t, float64(50), plan["fee"])
	assert.NotEmpty(t, plan["id"])
}

func TestCreatePlan_ValidationError(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	// Missing required name
	payload := map[string]any{
		"description": "No name plan",
		"fee":         10,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/billing/plans", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	data := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", data["status"])
}

func TestUpdatePlan_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	// Create a plan to update
	plan, err := models.CreatePlan(db.Postgresql, models.CreatePlanRequest{
		Name:        fmt.Sprintf("Update Test Plan %s", utility.RandomString(5)),
		Description: "Original description",
		Fee:         30,
		Benefits:    []string{"Original benefit"},
	})
	if err != nil {
		t.Fatalf("Failed to create test plan: %v", err)
	}

	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM plans WHERE id = ?", plan.ID)
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	newName := fmt.Sprintf("Updated Plan %s", utility.RandomString(5))
	newFee := 60
	payload := map[string]any{
		"name": newName,
		"fee":  newFee,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/backoffice/billing/plans/%s", plan.ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])

	updatedPlan, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("data['data'] is not a map, got: %T", data["data"])
	}

	assert.Equal(t, newName, updatedPlan["name"])
	assert.Equal(t, float64(newFee), updatedPlan["fee"])
}

func TestUpdatePlan_InvalidID(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	payload := map[string]any{"name": "whatever"}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/backoffice/billing/plans/not-a-uuid", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeletePlan_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	// Create a plan to delete
	plan, err := models.CreatePlan(db.Postgresql, models.CreatePlanRequest{
		Name:        fmt.Sprintf("Delete Test Plan %s", utility.RandomString(5)),
		Description: "To be deleted",
		Fee:         10,
	})
	if err != nil {
		t.Fatalf("Failed to create test plan: %v", err)
	}

	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM plans WHERE id = ?", plan.ID)
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/backoffice/billing/plans/%s", plan.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])
}

func TestDeletePlan_WithActiveSubscriptions(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)
	var createdOrgIDs []string

	// Create a plan and an org with an active subscription to it
	plan, err := models.CreatePlan(db.Postgresql, models.CreatePlanRequest{
		Name:        fmt.Sprintf("Active Sub Plan %s", utility.RandomString(5)),
		Description: "Has active subscriptions",
		Fee:         25,
	})
	if err != nil {
		t.Fatalf("Failed to create test plan: %v", err)
	}

	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 0)
	createdOrgIDs = append(createdOrgIDs, orgID)

	orgPlan := models.OrganisationPlan{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		PlanID:         plan.ID,
		Status:         "Active",
	}
	db.Postgresql.Create(&orgPlan)

	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM organisation_plans WHERE plan_id = ?", plan.ID)
		db.Postgresql.Exec("DELETE FROM plans WHERE id = ?", plan.ID)
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/backoffice/billing/plans/%s", plan.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	data := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", data["status"])
}

func TestGetPlansFiltered_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/billing/plans?status=active", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])
	assert.NotNil(t, data["data"])
}

func TestGetPlansFiltered_WithSearch(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/billing/plans?search=pro", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])

	plans, ok := data["data"].([]any)
	if ok {
		for _, p := range plans {
			plan := p.(map[string]any)
			name := plan["name"].(string)
			assert.Contains(t, []string{"Pro", "Pro Plus"}, name)
		}
	}
}

// --- AI Credit Package CRUD ---

func TestCreateAICreditPackage_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	pkgName := fmt.Sprintf("Test Package %s", utility.RandomString(5))

	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM ai_credit_packages WHERE name = ?", pkgName)
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	payload := models.CreateAICreditPackageRequest{
		Name:        pkgName,
		Description: "Test AI credit package",
		Price:       50,
		Credits:     5000,
		Benefits:    []string{"5000 AI credits", "Priority support"},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/billing/credit-packages", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusCreated, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])

	pkg, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("data['data'] is not a map, got: %T", data["data"])
	}

	assert.Equal(t, pkgName, pkg["name"])
	assert.Equal(t, float64(50), pkg["price"])
	assert.Equal(t, float64(5000), pkg["credits"])
	assert.NotEmpty(t, pkg["id"])
}

func TestCreateAICreditPackage_ValidationError(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	// Missing required fields
	payload := map[string]any{
		"description": "No name or price",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/billing/credit-packages", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateAICreditPackage_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	// Create a package to update
	pkg, err := models.CreateAICreditPackage(db.Postgresql, models.CreateAICreditPackageRequest{
		Name:        fmt.Sprintf("Update Pkg %s", utility.RandomString(5)),
		Description: "Original",
		Price:       30,
		Credits:     3000,
	})
	if err != nil {
		t.Fatalf("Failed to create test package: %v", err)
	}

	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM ai_credit_packages WHERE id = ?", pkg.ID)
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	newPrice := 45
	payload := map[string]any{
		"price": newPrice,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/backoffice/billing/credit-packages/%s", pkg.ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])

	updated, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("data['data'] is not a map, got: %T", data["data"])
	}

	assert.Equal(t, float64(newPrice), updated["price"])
}

func TestDeleteAICreditPackage_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	// Create a package to delete
	pkg, err := models.CreateAICreditPackage(db.Postgresql, models.CreateAICreditPackageRequest{
		Name:        fmt.Sprintf("Delete Pkg %s", utility.RandomString(5)),
		Description: "To be deleted",
		Price:       10,
		Credits:     1000,
	})
	if err != nil {
		t.Fatalf("Failed to create test package: %v", err)
	}

	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM ai_credit_packages WHERE id = ?", pkg.ID)
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/backoffice/billing/credit-packages/%s", pkg.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])
}

func TestDeleteAICreditPackage_InvalidID(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, nil)
	})

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/backoffice/billing/credit-packages/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// --- Transactions History ---

func TestGetAllCreditTransactions_Success(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)
	var createdOrgIDs []string

	// Create orgs with transactions
	org1ID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	createdOrgIDs = append(createdOrgIDs, org1ID)
	org2ID := CreateOrganizationWithCredit(t, db.Postgresql, 200.00)
	createdOrgIDs = append(createdOrgIDs, org2ID)

	CreateCreditTransaction(t, db.Postgresql, org1ID, 50.00)
	CreateCreditTransaction(t, db.Postgresql, org2ID, 75.00)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/billing/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])
	assert.NotNil(t, data["data"])
	assert.NotNil(t, data["pagination"])

	pagination, ok := data["pagination"].(map[string]any)
	if ok {
		assert.NotNil(t, pagination["total_items"])
		assert.GreaterOrEqual(t, pagination["total_items"].(float64), float64(2))
	}
}

func TestGetAllCreditTransactions_Unauthorized(t *testing.T) {
	_ = tst.Setup()
	r, _ := setupBillingTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/billing/transactions", nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetAllCreditTransactions_WithPagination(t *testing.T) {
	_ = tst.Setup()
	r, db := setupBillingTestRouter()

	adminID, token := CreateSuperAdminAndGetTokenWithID(t, r, db)
	var createdOrgIDs []string

	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 500.00)
	createdOrgIDs = append(createdOrgIDs, orgID)

	for i := 0; i < 5; i++ {
		CreateCreditTransaction(t, db.Postgresql, orgID, float64(i*10+10))
	}

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, adminID, createdOrgIDs)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/billing/transactions?page=1&page_size=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !assert.Equal(t, http.StatusOK, rr.Code) {
		t.Logf("Response body: %s", rr.Body.String())
		t.FailNow()
	}

	data := tst.ParseResponse(rr)
	assert.Equal(t, "success", data["status"])

	pagination, ok := data["pagination"].(map[string]any)
	if ok {
		assert.Equal(t, float64(2), pagination["page_size"])
	}
}
