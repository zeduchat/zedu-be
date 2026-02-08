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
	respData := data["data"].([]interface{})

	assert.Equal(t, 5, len(respData))

	foundPro := false
	for _, p := range respData {
		plan := p.(map[string]interface{})
		if plan["name"] == "Pro" {
			foundPro = true
			assert.Equal(t, "Ideal for growing learners", plan["description"])
			assert.Equal(t, 20.0, plan["fee"])
			assert.Equal(t, 60.0, plan["max_call_duration"])

			benefits := plan["benefits"].([]interface{})
			assert.Greater(t, len(benefits), 0)
			assert.Contains(t, benefits, "Advanced controls for administrator only")
		}
	}
	assert.True(t, foundPro, "Pro plan not found in response")
}
