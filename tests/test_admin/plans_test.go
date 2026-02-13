package test_admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/seed"
	"github.com/hngprojects/telex_be/pkg/controller/admin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func TestGetPlans(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	// Force migration to ensure new fields are present
	if err := db.Postgresql.AutoMigrate(&models.Plan{}); err != nil {
		t.Logf("Migration failed: %v", err)
	}

	// Seed plans
	seed.SeedPlans(logger, db.Postgresql)

	// Check what's actually in DB
	var allPlans []models.Plan
	db.Postgresql.Find(&allPlans)
	t.Logf("Found %d plans in DB after seeding:", len(allPlans))
	for _, p := range allPlans {
		t.Logf("- Name: %s, ID: %s, DeletedAt: %v", p.Name, p.ID, p.DeletedAt)
	}

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/plans", adminCtl.GetPlans)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/plans", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	data := tst.ParseResponse(rr)
	respMap, ok := data["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' to be a map, got %T. Response body: %s", data["data"], rr.Body.String())
	}

	plansData, ok := respMap["plans"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'plans' to be a slice, got %T", respMap["plans"])
	}

	assert.Equal(t, 5, len(plansData))

	foundPro := false
	for _, p := range plansData {
		plan := p.(map[string]interface{})
		if plan["name"] == "Pro" {
			foundPro = true
			assert.Equal(t, "Ideal for growing learners", plan["description"])
			assert.Equal(t, float64(20), plan["fee"]) // Changed to float64 for JSON unmarshaling

			benefits, ok := plan["benefits"].([]interface{})
			assert.True(t, ok)
			assert.Greater(t, len(benefits), 0)
			assert.Contains(t, benefits, "Advanced controls for administrator only")
		}
	}
	assert.True(t, foundPro, "Pro plan not found in response")

	aiPackages, ok := respMap["ai_credit_packages"].([]interface{})
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(aiPackages), 3) // Based on seeding
}
