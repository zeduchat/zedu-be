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
	"github.com/hngprojects/telex_be/pkg/controller/subscriptions"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreditTopupWebhook_Success(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()
	validator := validator.New()

	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	packageID := CreateTestCreditPackage(t, db.Postgresql, 1000, 5.00)
	sessionID := fmt.Sprintf("cs_test_%s", utility.GenerateUUID())

	webhookController := subscriptions.Controller{
		Db:        db,
		Validator: validator,
		Logger:    utility.NewLogger(),
		ExtReq:    request.ExternalRequest{},
	}

	r := gin.New()
	r.POST("/api/v1/subscriptions/webhook", webhookController.HandleStripeWebhook)

	webhookEvent := CreateTestStripeWebhookEvent(orgID, packageID, sessionID, "https://telex.im/client/settings/organisation/billing?session_id="+sessionID, "https://telex.im/client/settings/organisation/billing")

	eventJSON, _ := json.Marshal(webhookEvent)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscriptions/webhook", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test_signature")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])

	var transaction models.CreditTransaction
	err := db.Postgresql.Where("organisation_id = ? AND type = ?", orgID, "Top-up").First(&transaction).Error
	assert.NoError(t, err)
	assert.Equal(t, float64(1000), transaction.Amount)
	assert.Equal(t, float64(100.00), transaction.BalanceBefore)
	assert.Equal(t, float64(1100.00), transaction.BalanceAfter)

	var org models.Organisation
	err = db.Postgresql.Where("id = ?", orgID).First(&org).Error
	assert.NoError(t, err)
	assert.Equal(t, float64(1000.00), org.CreditBalance)
}

func TestCreditTopupWebhook_InvalidPackageID(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()
	validator := validator.New()

	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	sessionID := fmt.Sprintf("cs_test_%s", utility.GenerateUUID())

	webhookController := subscriptions.Controller{
		Db:        db,
		Validator: validator,
		Logger:    utility.NewLogger(),
		ExtReq:    request.ExternalRequest{},
	}

	r := gin.New()
	r.POST("/api/v1/subscriptions/webhook", webhookController.HandleStripeWebhook)

	webhookEvent := CreateTestStripeWebhookEvent(orgID, "invalid_package_id", sessionID, "https://telex.im/client/settings/organisation/billing?session_id="+sessionID, "https://telex.im/client/settings/organisation/billing")

	eventJSON, _ := json.Marshal(webhookEvent)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscriptions/webhook", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test_signature")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	response := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", response["status"])

	var transactionCount int64
	db.Postgresql.Model(&models.CreditTransaction{}).Where("organisation_id = ?", orgID).Count(&transactionCount)
	assert.Equal(t, int64(0), transactionCount)
}

func TestCreditTopupWebhook_InvalidOrgID(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()
	validator := validator.New()

	packageID := CreateTestCreditPackage(t, db.Postgresql, 1000, 5.00)
	sessionID := fmt.Sprintf("cs_test_%s", utility.GenerateUUID())

	webhookController := subscriptions.Controller{
		Db:        db,
		Validator: validator,
		Logger:    utility.NewLogger(),
		ExtReq:    request.ExternalRequest{},
	}

	r := gin.New()
	r.POST("/api/v1/subscriptions/webhook", webhookController.HandleStripeWebhook)

	webhookEvent := CreateTestStripeWebhookEvent("invalid_org_id", packageID, sessionID, "https://telex.im/client/settings/organisation/billing?session_id="+sessionID, "https://telex.im/client/settings/organisation/billing")

	eventJSON, _ := json.Marshal(webhookEvent)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscriptions/webhook", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test_signature")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusNotFound, rr.Code)

	response := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", response["status"])
}

func TestCreditTopupWebhook_WithExistingCreditUsage(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()
	validator := validator.New()

	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	packageID := CreateTestCreditPackage(t, db.Postgresql, 500, 10.00)
	sessionID := fmt.Sprintf("cs_test_%s", utility.GenerateUUID())

	CreateCreditUsage(t, db.Postgresql, orgID, 20.00)
	CreateCreditUsage(t, db.Postgresql, orgID, 30.00)

	webhookController := subscriptions.Controller{
		Db:        db,
		Validator: validator,
		Logger:    utility.NewLogger(),
		ExtReq:    request.ExternalRequest{},
	}

	r := gin.New()
	r.POST("/api/v1/subscriptions/webhook", webhookController.HandleStripeWebhook)

	webhookEvent := CreateTestStripeWebhookEvent(orgID, packageID, sessionID, "https://telex.im/client/settings/organisation/billing?session_id="+sessionID, "https://telex.im/client/settings/organisation/billing")

	eventJSON, _ := json.Marshal(webhookEvent)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscriptions/webhook", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test_signature")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	var transaction models.CreditTransaction
	err := db.Postgresql.Where("organisation_id = ?", orgID).Order("created_at DESC").First(&transaction).Error
	assert.NoError(t, err)
	assert.Equal(t, float64(500), transaction.Amount)
	assert.Equal(t, float64(100.00), transaction.BalanceBefore)
	assert.Equal(t, float64(600.00), transaction.BalanceAfter)

	var org models.Organisation
	err = db.Postgresql.Where("id = ?", orgID).First(&org).Error
	assert.NoError(t, err)
	assert.Equal(t, float64(450.00), org.CreditBalance)
}

func CreateTestStripeWebhookEvent(orgID, packageID, sessionID, successURL, cancelURL string) map[string]any {
	checkoutSession := map[string]any{
		"id":          sessionID,
		"object":      "checkout.session",
		"success_url": successURL,
		"cancel_url":  cancelURL,
		"metadata": map[string]string{
			"flow":       "credit_topup",
			"org_id":     orgID,
			"package_id": packageID,
		},
		"customer_details": map[string]any{
			"email": fmt.Sprintf("test%s@qa.team", utility.RandomString(5)),
		},
		"customer": map[string]any{
			"email": fmt.Sprintf("test%s@qa.team", utility.RandomString(5)),
		},
	}

	return map[string]any{
		"id":     fmt.Sprintf("evt_test_%s", utility.GenerateUUID()),
		"type":   "checkout.session.completed",
		"object": "event",
		"data": map[string]any{
			"object": checkoutSession,
		},
	}
}
