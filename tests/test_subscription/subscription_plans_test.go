package test_subscription

import (
	"encoding/json"
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
)

func TestGetSubscriptionPlans(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	// Check what's actually in DB
	var allPlans []models.Plan
	db.Postgresql.Find(&allPlans)
	t.Logf("Found %d plans in DB after seeding", len(allPlans))

	subCtl := subscriptions.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	r.GET("/api/v1/subscriptions/plans", subCtl.GetSubscriptionPlans)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscriptions/plans", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Logf("Response status: %d", rr.Code)
	t.Logf("Response body: %s", rr.Body.String())

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "success", response["status"])

	data := response["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 1, "Expected at least 1 plan")

	// Check that the first plan has expected fields
	firstPlan := data[0].(map[string]interface{})
	assert.NotEmpty(t, firstPlan["id"])
	assert.NotEmpty(t, firstPlan["name"])
}
