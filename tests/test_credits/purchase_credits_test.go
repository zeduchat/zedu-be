package test_credits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v72"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestPurchaseCredits_Success(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	stripeKey := config.Config.Stripe.STRIPE_KEY
	if stripeKey == "" {
		t.Skip("STRIPE_KEY not set, skipping test")
	}
	stripe.Key = stripeKey

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, email := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	pkg := GetSeededCreditPackage(t, db.Postgresql, "Starter Pack")

	requestBody := models.CreditTopUpRequest{
		OrgID:     orgID,
		PackageID: pkg.ID,
		Email:     email,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Referer", "https://telex.im/")
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Logf("Response body: %s", rr.Body.String())
	}

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])
	assert.NotNil(t, response["data"])

	data, ok := response["data"].(map[string]any)
	assert.True(t, ok, "data should be a map")

	assert.NotEmpty(t, data["checkout_session_id"], "checkout_session_id should not be empty")
	assert.NotEmpty(t, data["checkout_session_url"], "checkout_session_url should not be empty")

	sessionURL, ok := data["checkout_session_url"].(string)
	assert.True(t, ok, "checkout_session_url should be a string")
	assert.Contains(t, sessionURL, "checkout.stripe.com", "should be a valid Stripe checkout URL")
}

func TestPurchaseCredits_WithProBundle(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	stripeKey := config.Config.Stripe.STRIPE_KEY
	if stripeKey == "" {
		t.Skip("STRIPE_KEY not set, skipping test")
	}
	stripe.Key = stripeKey

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, email := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	pkg := GetSeededCreditPackage(t, db.Postgresql, "Pro Bundle")

	requestBody := models.CreditTopUpRequest{
		OrgID:     orgID,
		PackageID: pkg.ID,
		Email:     email,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Referer", "https://telex.im/")
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])
	assert.NotNil(t, response["data"])

	data, ok := response["data"].(map[string]any)
	assert.True(t, ok)
	assert.NotEmpty(t, data["checkout_session_id"])
	assert.NotEmpty(t, data["checkout_session_url"])
}

func TestPurchaseCredits_InvalidPackageID(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	stripeKey := config.Config.Stripe.STRIPE_KEY
	if stripeKey == "" {
		t.Skip("STRIPE_KEY not set, skipping test")
	}
	stripe.Key = stripeKey

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, email := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	requestBody := models.CreditTopUpRequest{
		OrgID:     orgID,
		PackageID: "00000000-0000-0000-0000-000000000000",
		Email:     email,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Referer", "https://telex.im/")
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)

	response := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", response["status"])
}

func TestPurchaseCredits_Unauthorized(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, _, _, _, _ := SetupCreditTestRouter()

	requestBody := models.CreditTopUpRequest{
		PackageID: utility.GenerateUUID(),
		Email:     "test@example.com",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://telex.im/")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	response := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", response["status"])
}

func TestPurchaseCredits_MissingReferer(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	stripeKey := config.Config.Stripe.STRIPE_KEY
	if stripeKey == "" {
		t.Skip("STRIPE_KEY not set, skipping test")
	}
	stripe.Key = stripeKey

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, email := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	pkg := GetSeededCreditPackage(t, db.Postgresql, "Starter Pack")

	requestBody := models.CreditTopUpRequest{
		OrgID:     orgID,
		PackageID: pkg.ID,
		Email:     email,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	response := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", response["status"])
	assert.Contains(t, fmt.Sprintf("%v", response["message"]), "missing URL")
}

func TestPurchaseCredits_MissingOrgContext(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	stripeKey := config.Config.Stripe.STRIPE_KEY
	if stripeKey == "" {
		t.Skip("STRIPE_KEY not set, skipping test")
	}
	stripe.Key = stripeKey

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, email := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	pkg := GetSeededCreditPackage(t, db.Postgresql, "Starter Pack")

	requestBody := models.CreditTopUpRequest{
		PackageID: pkg.ID,
		Email:     email,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Referer", "https://telex.im/")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])
}

func TestPurchaseCredits_InvalidRequestBody(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, _ := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Referer", "https://telex.im/")
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	response := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", response["status"])
}
