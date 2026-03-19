package test_admin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/admin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func setupUserDetailsRouter(adminCtl admin.Controller) *gin.Engine {
	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users/:user_id/details", adminCtl.GetUserDetails)
	return r
}

func createTestUser(t *testing.T, db *storage.Database) models.User {
	t.Helper()
	userID := utility.GenerateUUID()
	now := time.Now()
	password, _ := utility.HashPassword("testpassword")

	user := models.User{
		ID:         userID,
		Name:       fmt.Sprintf("TestUser_%s", utility.RandomString(5)),
		Email:      fmt.Sprintf("testuser%s@qa.team", utility.RandomString(5)),
		Password:   password,
		IsActive:   true,
		IsVerified: true,
		CreatedAt:  now,
		Profile: models.Profile{
			ID:        utility.GenerateUUID(),
			Userid:    userID,
			FirstName: "Test",
			LastName:  "User",
			AvatarURL: "",
		},
	}

	if err := db.Postgresql.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM profiles WHERE userid = ?", userID)
		db.Postgresql.Exec("DELETE FROM users WHERE id = ?", userID)
	})

	return user
}

func TestGetUserDetails_ReturnsOK(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	user := createTestUser(t, db)

	r := setupUserDetailsRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users/%s/details", user.ID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	if _, ok := respData["profile"]; !ok {
		t.Error("expected profile field in response")
	}
	if _, ok := respData["credit_stats"]; !ok {
		t.Error("expected credit_stats field in response")
	}
	if _, ok := respData["activity_info"]; !ok {
		t.Error("expected activity_info field in response")
	}
	if _, ok := respData["app_usage"]; !ok {
		t.Error("expected app_usage field in response")
	}
}

func TestGetUserDetails_UserNotFound(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	fakeID := utility.GenerateUUID()

	r := setupUserDetailsRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users/%s/details", fakeID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
}

func TestGetUserDetails_ProfileShape(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	user := createTestUser(t, db)

	r := setupUserDetailsRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users/%s/details", user.ID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	profile := respData["profile"].(map[string]any)

	expectedFields := []string{"id", "name", "email", "avatar_url", "default_avatar_url", "subscription_status"}
	for _, field := range expectedFields {
		if _, ok := profile[field]; !ok {
			t.Errorf("missing expected profile field: %s", field)
		}
	}

	if profile["id"] != user.ID {
		t.Errorf("expected profile id=%s, got %v", user.ID, profile["id"])
	}
	if profile["email"] != user.Email {
		t.Errorf("expected profile email=%s, got %v", user.Email, profile["email"])
	}
	if profile["subscription_status"] != "Free" {
		t.Errorf("expected subscription_status=Free, got %v", profile["subscription_status"])
	}
}

func TestGetUserDetails_AppUsageShape(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	user := createTestUser(t, db)

	r := setupUserDetailsRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users/%s/details", user.ID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	appUsage := respData["app_usage"].(map[string]any)

	expectedFields := []string{
		"total_items_created", "total_items_created_change",
		"sessions_initiated", "sessions_initiated_change",
		"avg_session_duration", "avg_session_duration_change",
		"key_actions_performed", "key_actions_change",
	}
	for _, field := range expectedFields {
		if _, ok := appUsage[field]; !ok {
			t.Errorf("missing expected app_usage field: %s", field)
		}
	}
}

func TestGetUserDetails_CreditStatsShape(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	user := createTestUser(t, db)

	r := setupUserDetailsRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users/%s/details", user.ID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	creditStats := respData["credit_stats"].(map[string]any)

	expectedFields := []string{
		"total_credits_used", "credits_used_change",
		"total_amount_spent", "amount_spent_change",
	}
	for _, field := range expectedFields {
		if _, ok := creditStats[field]; !ok {
			t.Errorf("missing expected credit_stats field: %s", field)
		}
	}
}

func TestGetUserDetails_ActivityInfoShape(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	user := createTestUser(t, db)

	r := setupUserDetailsRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users/%s/details", user.ID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activityInfo := respData["activity_info"].(map[string]any)

	expectedFields := []string{"last_active", "last_login", "activity_length", "referrals"}
	for _, field := range expectedFields {
		if _, ok := activityInfo[field]; !ok {
			t.Errorf("missing expected activity_info field: %s", field)
		}
	}
}

func TestGetUserDetails_WithAuditLogs(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	user := createTestUser(t, db)
	now := time.Now()

	// Create some audit logs for the user
	for i := 0; i < 3; i++ {
		createAuditLog(t, db, user.ID, user.Email, "user", "user.login", now.AddDate(0, 0, -i), true)
	}

	r := setupUserDetailsRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users/%s/details", user.ID), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	appUsage := respData["app_usage"].(map[string]any)

	if appUsage["key_actions_performed"].(float64) < 3 {
		t.Errorf("expected key_actions_performed >= 3, got %v", appUsage["key_actions_performed"])
	}
}
