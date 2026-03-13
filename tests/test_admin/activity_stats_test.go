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

func setupActivityRouter(adminCtl admin.Controller) *gin.Engine {
	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/activity", adminCtl.GetAppActivity)
	return r
}

func createAuditLog(t *testing.T, db *storage.Database, actorID, actorEmail, actorRole, action string, createdAt time.Time, success bool) string {
	t.Helper()
	logID := utility.GenerateUUID()
	log := models.AuditLog{
		ID:           logID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		ActorRole:    actorRole,
		Action:       models.AuditAction(action),
		ResourceType: models.ResourceType("user"),
		ResourceID:   "",
		OldValues:    "{}",
		NewValues:    "{}",
		Description:  fmt.Sprintf("Actor %s performed %s", actorEmail, action),
		IPAddress:    "",
		UserAgent:    "",
		Success:      true, // placeholder; overridden below via raw UPDATE
		CreatedAt:    createdAt,
	}
	if err := db.Postgresql.Create(&log).Error; err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}
	// GORM silently skips bool false (zero value) during Create, so we force
	// both success and created_at with a raw UPDATE after the insert.
	db.Postgresql.Exec(
		"UPDATE audit_logs SET success = ?, created_at = ? WHERE id = ?",
		success, createdAt, logID,
	)
	return logID
}

func TestGetAppActivity_ReturnsOK(t *testing.T) {
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

	actorID := utility.GenerateUUID()
	now := time.Now()
	createAuditLog(t, db, actorID, fmt.Sprintf("actor%s@qa.team", utility.RandomString(5)), "admin", "user.login", now, true)

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/activity", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	if _, ok := respData["activity_stats"]; !ok {
		t.Error("expected activity_stats field in response")
	}
	if _, ok := respData["activities"]; !ok {
		t.Error("expected activities field in response")
	}
}

func TestGetAppActivity_IncludesStats(t *testing.T) {
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

	now := time.Now()
	actorID := utility.GenerateUUID()
	email := fmt.Sprintf("statsactor%s@qa.team", utility.RandomString(5))

	// Create logs in last 30 days
	createAuditLog(t, db, actorID, email, "admin", "user.create", now.AddDate(0, 0, -5), true)
	createAuditLog(t, db, actorID, email, "admin", "user.update", now.AddDate(0, 0, -10), true)
	createAuditLog(t, db, actorID, email, "user", "user.login", now.AddDate(0, 0, -2), true)
	// failed_login must have Success: false so the stats query counts it as a failed action
	createAuditLog(t, db, actorID, email, "user", "failed_login", now.AddDate(0, 0, -1), false)

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/activity?include_stats=true", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["activity_stats"].(map[string]any)

	if stats["total_activities_month"].(float64) < 1 {
		t.Errorf("expected total_activities_month >= 1, got %v", stats["total_activities_month"])
	}
	if stats["admin_actions_month"].(float64) < 1 {
		t.Errorf("expected admin_actions_month >= 1, got %v", stats["admin_actions_month"])
	}
	if stats["user_actions_month"].(float64) < 1 {
		t.Errorf("expected user_actions_month >= 1, got %v", stats["user_actions_month"])
	}
	if stats["failed_actions_month"].(float64) < 1 {
		t.Errorf("expected failed_actions_month >= 1, got %v", stats["failed_actions_month"])
	}
}

func TestGetAppActivity_FilterByRole(t *testing.T) {
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

	now := time.Now()
	suffix := utility.RandomString(6)
	adminEmail := fmt.Sprintf("adminrole%s@qa.team", suffix)
	userEmail := fmt.Sprintf("userrole%s@qa.team", suffix)
	actorID := utility.GenerateUUID()

	createAuditLog(t, db, actorID, adminEmail, "admin", "user.create", now, true)
	createAuditLog(t, db, utility.GenerateUUID(), userEmail, "user", "user.login", now, true)

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/activity?role=admin", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activities := respData["activities"].([]any)

	for _, a := range activities {
		act := a.(map[string]any)
		if act["role"] != "admin" {
			t.Errorf("expected role=admin, got %v", act["role"])
		}
	}
}

func TestGetAppActivity_FilterByStatus_Failed(t *testing.T) {
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

	now := time.Now()
	suffix := utility.RandomString(6)
	email := fmt.Sprintf("failedactor%s@qa.team", suffix)
	actorID := utility.GenerateUUID()

	// Success: false so the record is counted/filtered as a failed action
	createAuditLog(t, db, actorID, email, "user", "failed_login", now, false)
	createAuditLog(t, db, actorID, email, "user", "user.login", now, true)

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/activity?status=failed", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activities := respData["activities"].([]any)

	for _, a := range activities {
		act := a.(map[string]any)
		if act["status"] != "failed" {
			t.Errorf("expected status=failed, got %v", act["status"])
		}
	}
}

func TestGetAppActivity_FilterByStatus_Success(t *testing.T) {
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

	now := time.Now()
	suffix := utility.RandomString(6)
	email := fmt.Sprintf("successactor%s@qa.team", suffix)
	actorID := utility.GenerateUUID()

	createAuditLog(t, db, actorID, email, "user", "user.login", now, true)
	createAuditLog(t, db, actorID, email, "user", "failed_login", now, false)

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/activity?status=success", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activities := respData["activities"].([]any)

	for _, a := range activities {
		act := a.(map[string]any)
		if act["status"] != "success" {
			t.Errorf("expected status=success, got %v", act["status"])
		}
	}
}

func TestGetAppActivity_FilterBySearch(t *testing.T) {
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

	now := time.Now()
	suffix := utility.RandomString(6)
	uniqueEmail := fmt.Sprintf("searchable%s@qa.team", suffix)
	actorID := utility.GenerateUUID()

	// Seed a real admin record so the service's actor join populates user.email
	password, _ := utility.HashPassword("password")
	adminRecord := models.Admin{
		ID:       actorID,
		Name:     "Search Actor",
		Email:    uniqueEmail,
		Role:     models.RoleAdmin,
		IsActive: true,
		Password: password,
	}
	if err := db.Postgresql.Create(&adminRecord).Error; err != nil {
		t.Fatalf("failed to create admin record for search test: %v", err)
	}
	t.Cleanup(func() {
		db.Postgresql.Exec("DELETE FROM admins WHERE id = ?", actorID)
	})

	createAuditLog(t, db, actorID, uniqueEmail, "admin", "user.create", now, true)

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/activity?search=%s", uniqueEmail), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activities := respData["activities"].([]any)

	if len(activities) == 0 {
		t.Fatalf("expected at least 1 result for search=%s, got 0", uniqueEmail)
	}

	found := false
	for _, a := range activities {
		act := a.(map[string]any)
		user := act["user"].(map[string]any)
		if user["email"] == uniqueEmail {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find activity with email %s in results", uniqueEmail)
	}
}

func TestGetAppActivity_FilterByDuration_LastMonth(t *testing.T) {
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

	now := time.Now()
	suffix := utility.RandomString(6)
	email := fmt.Sprintf("durationactor%s@qa.team", suffix)
	actorID := utility.GenerateUUID()

	// recent log (within last month)
	createAuditLog(t, db, actorID, email, "admin", "user.create", now.AddDate(0, 0, -15), true)
	// old log (older than 1 month)
	oldID := createAuditLog(t, db, actorID, email, "admin", "user.delete", now.AddDate(0, -2, 0), true)
	db.Postgresql.Model(&models.AuditLog{}).Where("id = ?", oldID).Update("created_at", now.AddDate(0, -2, 0))

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/activity?duration=last_month", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activities := respData["activities"].([]any)

	for _, a := range activities {
		act := a.(map[string]any)
		details := act["details"].(map[string]any)
		ts := details["timestamp"].(string)
		parsed, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			parsed, _ = time.Parse("2006-01-02T15:04:05Z07:00", ts)
		}
		cutoff := now.AddDate(0, -1, 0)
		if parsed.Before(cutoff) {
			t.Errorf("activity timestamp %v is older than last_month cutoff %v", parsed, cutoff)
		}
	}
}

func TestGetAppActivity_NoStats_WhenExcluded(t *testing.T) {
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

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/activity?include_stats=false", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["activity_stats"].(map[string]any)

	// When stats are excluded, all stat values should be zero
	if stats["total_activities_month"].(float64) != 0 {
		t.Errorf("expected total_activities_month=0 when stats excluded, got %v", stats["total_activities_month"])
	}
}

func TestGetAppActivity_ActivityShape(t *testing.T) {
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

	now := time.Now()
	suffix := utility.RandomString(6)
	email := fmt.Sprintf("shapeactor%s@qa.team", suffix)
	actorID := utility.GenerateUUID()

	createAuditLog(t, db, actorID, email, "admin", "user.create", now, true)

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/activity?search=%s", email), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activities := respData["activities"].([]any)

	if len(activities) == 0 {
		t.Fatal("expected at least 1 activity")
	}

	act := activities[0].(map[string]any)

	requiredFields := []string{"activity_id", "user", "role", "action", "resource", "status", "details"}
	for _, field := range requiredFields {
		if _, ok := act[field]; !ok {
			t.Errorf("missing expected field: %s", field)
		}
	}

	user := act["user"].(map[string]any)
	userFields := []string{"id", "name", "email", "avatar_url"}
	for _, field := range userFields {
		if _, ok := user[field]; !ok {
			t.Errorf("missing expected user field: %s", field)
		}
	}

	details := act["details"].(map[string]any)
	detailFields := []string{"description", "timestamp"}
	for _, field := range detailFields {
		if _, ok := details[field]; !ok {
			t.Errorf("missing expected details field: %s", field)
		}
	}
}

func TestGetAppActivity_Pagination(t *testing.T) {
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

	now := time.Now()
	suffix := utility.RandomString(6)
	email := fmt.Sprintf("pageactor%s@qa.team", suffix)
	actorID := utility.GenerateUUID()

	for i := 0; i < 5; i++ {
		createAuditLog(t, db, actorID, email, "admin", fmt.Sprintf("user.action%d", i), now.Add(time.Duration(-i)*time.Minute), true)
	}

	r := setupActivityRouter(adminCtl)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/activity?search=%s&page=1&limit=2", email), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	activities := respData["activities"].([]any)

	if len(activities) > 2 {
		t.Errorf("expected at most 2 activities per page, got %d", len(activities))
	}

	if _, ok := data["pagination"]; !ok {
		t.Error("expected pagination field in response")
	}
}
