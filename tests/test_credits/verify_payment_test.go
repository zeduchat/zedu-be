package test_credits

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/stripe/stripe-go/v72"
)

func TestVerifyPayment_Unauthorized(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, _, _, _, _ := SetupCreditTestRouter()

	requestBody := models.VerifyPaymentRequest{
		SessionID: "cs_test_123",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/verify-payment", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestVerifyPayment_InvalidSession(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	stripeKey := config.Config.Stripe.STRIPE_KEY
	if stripeKey != "" {
		stripe.Key = stripeKey
	}

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, _ := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	requestBody := models.VerifyPaymentRequest{
		SessionID: "cs_test_a1yvVa2yYQw4Mm0JOVbRWWtjdIRcYikZTPMFWDegPDmKmlUU9gEHLAwDW5",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/credits/verify-payment", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusNotFound)

	response := tst.ParseResponse(rr)
	if rr.Code == http.StatusNotFound {
		assert.Contains(t, response["message"], "org not found")
	} else {
		assert.Contains(t, response["message"], "failed to retrieve stripe session")
	}
}
