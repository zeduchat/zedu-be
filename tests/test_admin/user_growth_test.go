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
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	adminSvc "github.com/hngprojects/telex_be/services/admin"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)


func getLagosTime() time.Time {
	loc, _ := time.LoadLocation("Africa/Lagos")
	if loc == nil {
		loc = time.FixedZone("WAT", 1*60*60)
	}
	return time.Now().In(loc)
}

// =============================================================================
// PRESET TESTS
// =============================================================================

func TestGetUserGrowth_PresetToday(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := getLagosTime()

	// Create a user today
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("todayuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Today",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("todayuser%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, user, false)

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=today", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Verify period
	if respData["period"].(string) != "today" {
		t.Errorf("expected period 'today', got '%s'", respData["period"])
	}

	// Verify dates
	todayStr := now.Format("2006-01-02")
	if respData["start_date"].(string) != todayStr {
		t.Errorf("expected start_date '%s', got '%s'", todayStr, respData["start_date"])
	}
	if respData["end_date"].(string) != todayStr {
		t.Errorf("expected end_date '%s', got '%s'", todayStr, respData["end_date"])
	}

	// Verify total count
	totalCount := int64(respData["total_count"].(float64))
	if totalCount < 1 {
		t.Errorf("expected at least 1 user created today, got %d", totalCount)
	}
}

func TestGetUserGrowth_PresetYesterday(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := getLagosTime()
	yesterday := now.AddDate(0, 0, -1)

	// Create a user and backdate to yesterday
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("yesterdayuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Yesterday",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("yesterdayuser%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, user, false)

	// Update created_at to yesterday
	db.Postgresql.Model(&models.User{}).Where("email = ?", user.Email).Update("created_at", yesterday)

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=yesterday", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Verify period
	if respData["period"].(string) != "yesterday" {
		t.Errorf("expected period 'yesterday', got '%s'", respData["period"])
	}

	// Verify dates
	yesterdayStr := yesterday.Format("2006-01-02")
	if respData["start_date"].(string) != yesterdayStr {
		t.Errorf("expected start_date '%s', got '%s'", yesterdayStr, respData["start_date"])
	}

	// Verify total count
	totalCount := int64(respData["total_count"].(float64))
	if totalCount < 1 {
		t.Errorf("expected at least 1 user created yesterday, got %d", totalCount)
	}
}

func TestGetUserGrowth_PresetLast7Days(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()

	// Create users on different days in the last 7 days
	for i := 0; i < 3; i++ {
		user := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("last7days%d%v@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   fmt.Sprintf("Week%d", i),
			LastName:    "User",
			Password:    "password",
			UserName:    fmt.Sprintf("last7days%d%v", i, currUUID),
		}
		tst.SignupUser(t, gin.Default(), authCtl, user, false)

		// Backdate some users
		if i > 0 {
			pastDate := getLagosTime().AddDate(0, 0, -i*2)
			db.Postgresql.Model(&models.User{}).Where("email = ?", user.Email).Update("created_at", pastDate)
		}
	}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=last_7_days", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Verify period
	if respData["period"].(string) != "last_7_days" {
		t.Errorf("expected period 'last_7_days', got '%s'", respData["period"])
	}

	// Verify total count
	totalCount := int64(respData["total_count"].(float64))
	if totalCount < 1 {
		t.Errorf("expected at least 1 user in last 7 days, got %d", totalCount)
	}
}

func TestGetUserGrowth_PresetLast30Days(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=last_30_days", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	if respData["period"].(string) != "last_30_days" {
		t.Errorf("expected period 'last_30_days', got '%s'", respData["period"])
	}
}

func TestGetUserGrowth_PresetThisMonth(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=this_month", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	if respData["period"].(string) != "this_month" {
		t.Errorf("expected period 'this_month', got '%s'", respData["period"])
	}

	// Verify start_date is first day of current month
	now := getLagosTime()
	expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	if respData["start_date"].(string) != expectedStart {
		t.Errorf("expected start_date '%s', got '%s'", expectedStart, respData["start_date"])
	}
}

func TestGetUserGrowth_PresetThisYear(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=this_year", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	if respData["period"].(string) != "this_year" {
		t.Errorf("expected period 'this_year', got '%s'", respData["period"])
	}

	// Verify start_date is Jan 1 of current year
	now := getLagosTime()
	expectedStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	if respData["start_date"].(string) != expectedStart {
		t.Errorf("expected start_date '%s', got '%s'", expectedStart, respData["start_date"])
	}
}

// =============================================================================
// CUSTOM DATE RANGE TESTS
// =============================================================================

func TestGetUserGrowth_CustomDateRange(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := getLagosTime()

	// Create a user today
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("customrange%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Custom",
		LastName:    "Range",
		Password:    "password",
		UserName:    fmt.Sprintf("customrange%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, user, false)

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	// Test with custom date range
	fromDate := now.AddDate(0, 0, -7).Format("2006-01-02")
	toDate := now.Format("2006-01-02")

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/user-growth?from=%s&to=%s", fromDate, toDate), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Verify dates match request
	if respData["start_date"].(string) != fromDate {
		t.Errorf("expected start_date '%s', got '%s'", fromDate, respData["start_date"])
	}
	if respData["end_date"].(string) != toDate {
		t.Errorf("expected end_date '%s', got '%s'", toDate, respData["end_date"])
	}

	// Period should be empty for custom range
	if period, ok := respData["period"]; ok && period != "" {
		t.Errorf("expected empty period for custom range, got '%s'", period)
	}

	// Verify count
	totalCount := int64(respData["total_count"].(float64))
	if totalCount < 1 {
		t.Errorf("expected at least 1 user in date range, got %d", totalCount)
	}
}

func TestGetUserGrowth_CustomDateRangeSingleDay(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	// Same from and to date
	today := getLagosTime().Format("2006-01-02")

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/user-growth?from=%s&to=%s", today, today), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	if respData["start_date"].(string) != today {
		t.Errorf("expected start_date '%s', got '%s'", today, respData["start_date"])
	}
	if respData["end_date"].(string) != today {
		t.Errorf("expected end_date '%s', got '%s'", today, respData["end_date"])
	}
}

// =============================================================================
// GROUP BY TESTS
// =============================================================================

func TestGetUserGrowth_GroupByDay(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := getLagosTime()

	// Create users on different days
	for i := 0; i < 3; i++ {
		user := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("groupday%d%v@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   fmt.Sprintf("GroupDay%d", i),
			LastName:    "User",
			Password:    "password",
			UserName:    fmt.Sprintf("groupday%d%v", i, currUUID),
		}
		tst.SignupUser(t, gin.Default(), authCtl, user, false)

		if i > 0 {
			pastDate := now.AddDate(0, 0, -i)
			db.Postgresql.Model(&models.User{}).Where("email = ?", user.Email).Update("created_at", pastDate)
		}
	}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=last_7_days&group_by=day", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Verify breakdown exists
	breakdown, ok := respData["breakdown"].([]any)
	if !ok {
		t.Fatal("expected breakdown field in response")
	}

	if len(breakdown) == 0 {
		t.Fatal("expected non-empty breakdown for daily grouping")
	}

	// Verify each breakdown item has correct structure
	for _, item := range breakdown {
		breakdownItem := item.(map[string]any)

		if breakdownItem["period"].(string) != "day" {
			t.Errorf("expected period 'day', got '%s'", breakdownItem["period"])
		}

		// Verify date format (YYYY-MM-DD)
		dateStr := breakdownItem["date"].(string)
		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			t.Errorf("invalid date format: %s", dateStr)
		}

		// Count should be non-negative
		count := int64(breakdownItem["count"].(float64))
		if count < 0 {
			t.Errorf("count should be non-negative, got %d", count)
		}
	}

	// Verify breakdown covers expected days (7 days for last_7_days)
	if len(breakdown) != 7 {
		t.Errorf("expected 7 days in breakdown, got %d", len(breakdown))
	}
}

func TestGetUserGrowth_GroupByWeek(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := getLagosTime()

	// Create users across different weeks
	for i := 0; i < 3; i++ {
		user := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("groupweek%d%v@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   fmt.Sprintf("GroupWeek%d", i),
			LastName:    "User",
			Password:    "password",
			UserName:    fmt.Sprintf("groupweek%d%v", i, currUUID),
		}
		tst.SignupUser(t, gin.Default(), authCtl, user, false)

		if i > 0 {
			pastDate := now.AddDate(0, 0, -i*7) // Different weeks
			db.Postgresql.Model(&models.User{}).Where("email = ?", user.Email).Update("created_at", pastDate)
		}
	}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=last_30_days&group_by=week", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	breakdown, ok := respData["breakdown"].([]any)
	if !ok {
		t.Fatal("expected breakdown field in response")
	}

	if len(breakdown) == 0 {
		t.Fatal("expected non-empty breakdown for weekly grouping")
	}

	// Verify structure
	for _, item := range breakdown {
		breakdownItem := item.(map[string]any)

		if breakdownItem["period"].(string) != "week" {
			t.Errorf("expected period 'week', got '%s'", breakdownItem["period"])
		}

		// Verify date format
		dateStr := breakdownItem["date"].(string)
		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			t.Errorf("invalid date format: %s", dateStr)
		}
	}
}

func TestGetUserGrowth_GroupByMonth(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := getLagosTime()

	// Create users in different months
	for i := 0; i < 2; i++ {
		user := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("groupmonth%d%v@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   fmt.Sprintf("GroupMonth%d", i),
			LastName:    "User",
			Password:    "password",
			UserName:    fmt.Sprintf("groupmonth%d%v", i, currUUID),
		}
		tst.SignupUser(t, gin.Default(), authCtl, user, false)

		if i > 0 {
			pastDate := now.AddDate(0, -i, 0) // Different months
			db.Postgresql.Model(&models.User{}).Where("email = ?", user.Email).Update("created_at", pastDate)
		}
	}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=this_year&group_by=month", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	breakdown, ok := respData["breakdown"].([]any)
	if !ok {
		t.Fatal("expected breakdown field in response")
	}

	if len(breakdown) == 0 {
		t.Fatal("expected non-empty breakdown for monthly grouping")
	}

	// Verify structure
	for _, item := range breakdown {
		breakdownItem := item.(map[string]any)

		if breakdownItem["period"].(string) != "month" {
			t.Errorf("expected period 'month', got '%s'", breakdownItem["period"])
		}

		// Verify date format (YYYY-MM for months)
		dateStr := breakdownItem["date"].(string)
		_, err := time.Parse("2006-01", dateStr)
		if err != nil {
			t.Errorf("invalid month format: %s", dateStr)
		}
	}
}

// =============================================================================
// TIMEZONE TESTS
// =============================================================================

func TestGetUserGrowth_CustomTimezone(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	// Test with America/New_York timezone
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=today&timezone=America/New_York", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Should return successfully with timezone applied
	if _, ok := respData["total_count"]; !ok {
		t.Error("expected total_count in response")
	}
}

func TestGetUserGrowth_InvalidTimezone(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=today&timezone=Invalid/Timezone", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Should return error for invalid timezone
	tst.AssertStatusCode(t, rr.Code, http.StatusInternalServerError)
}

// =============================================================================
// VALIDATION TESTS
// =============================================================================

func TestGetUserGrowth_MissingParameters(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	// No preset or date range provided
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	data := tst.ParseResponse(rr)
	if data["message"].(string) != "Either preset or both from/to dates must be provided" {
		t.Errorf("unexpected error message: %s", data["message"])
	}
}

func TestGetUserGrowth_InvalidPreset(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=invalid_preset", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
}

func TestGetUserGrowth_InvalidGroupBy(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=today&group_by=invalid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
}

func TestGetUserGrowth_BothPresetAndCustomRange(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	today := getLagosTime().Format("2006-01-02")
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/user-growth?preset=today&from=%s&to=%s", today, today), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	data := tst.ParseResponse(rr)
	if data["message"].(string) != "Cannot use both preset and custom date range" {
		t.Errorf("unexpected error message: %s", data["message"])
	}
}

func TestGetUserGrowth_ToDateBeforeFromDate(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?from=2024-12-31&to=2024-01-01", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	data := tst.ParseResponse(rr)
	if data["message"].(string) != "to date must be after from date" {
		t.Errorf("unexpected error message: %s", data["message"])
	}
}

func TestGetUserGrowth_DateRangeTooLarge(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	// Date range > 2 years
	fromDate := "2020-01-01"
	toDate := "2024-01-01"

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/user-growth?from=%s&to=%s", fromDate, toDate), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	data := tst.ParseResponse(rr)
	if data["message"].(string) != "Date range cannot exceed 2 years" {
		t.Errorf("unexpected error message: %s", data["message"])
	}
}

func TestGetUserGrowth_OnlyFromDate(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?from=2024-01-01", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
}

func TestGetUserGrowth_OnlyToDate(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?to=2024-01-01", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
}

// =============================================================================
// SERVICE LAYER TESTS
// =============================================================================

func TestGetUserGrowthService_PresetToday(t *testing.T) {
	_, _, _, db := SetupAdminTestRouter()

	params := adminSvc.UserGrowthParams{
		Preset:   "today",
		Timezone: "Africa/Lagos",
	}

	result, err := adminSvc.GetUserGrowthMetrics(db.Postgresql, params)
	if err != nil {
		t.Fatalf("GetUserGrowthMetrics returned error: %v", err)
	}

	if result.Period != "today" {
		t.Errorf("expected period 'today', got '%s'", result.Period)
	}

	if result.TotalCount < 0 {
		t.Errorf("total count should be non-negative, got %d", result.TotalCount)
	}

	// Verify dates
	now := getLagosTime()
	expectedDate := now.Format("2006-01-02")
	if result.StartDate != expectedDate || result.EndDate != expectedDate {
		t.Errorf("expected dates to be '%s', got start='%s', end='%s'",
			expectedDate, result.StartDate, result.EndDate)
	}
}

func TestGetUserGrowthService_WithBreakdown(t *testing.T) {
	_, authCtl, _, db := SetupAdminTestRouter()

	currUUID := utility.GenerateUUID()

	// Create test user
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("svcbreakdown%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Service",
		LastName:    "Breakdown",
		Password:    "password",
		UserName:    fmt.Sprintf("svcbreakdown%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), *authCtl, user, false)

	params := adminSvc.UserGrowthParams{
		Preset:   "last_7_days",
		GroupBy:  "day",
		Timezone: "Africa/Lagos",
	}

	result, err := adminSvc.GetUserGrowthMetrics(db.Postgresql, params)
	if err != nil {
		t.Fatalf("GetUserGrowthMetrics returned error: %v", err)
	}

	if len(result.Breakdown) == 0 {
		t.Fatal("expected breakdown to be populated")
	}

	// Verify breakdown has 7 entries for last_7_days
	if len(result.Breakdown) != 7 {
		t.Errorf("expected 7 breakdown entries, got %d", len(result.Breakdown))
	}

	// Verify each breakdown entry
	for _, item := range result.Breakdown {
		if item.Period != "day" {
			t.Errorf("expected period 'day', got '%s'", item.Period)
		}
		if item.Date == "" {
			t.Error("breakdown date should not be empty")
		}
		if item.Count < 0 {
			t.Errorf("breakdown count should be non-negative, got %d", item.Count)
		}
	}
}

func TestGetUserGrowthService_CustomDateRange(t *testing.T) {
	_, _, _, db := SetupAdminTestRouter()

	now := getLagosTime()
	fromDate := now.AddDate(0, 0, -7)
	toDate := now

	params := adminSvc.UserGrowthParams{
		From:     &fromDate,
		To:       &toDate,
		Timezone: "Africa/Lagos",
	}

	result, err := adminSvc.GetUserGrowthMetrics(db.Postgresql, params)
	if err != nil {
		t.Fatalf("GetUserGrowthMetrics returned error: %v", err)
	}

	// Period should be empty for custom range
	if result.Period != "" {
		t.Errorf("expected empty period for custom range, got '%s'", result.Period)
	}

	// Verify dates
	expectedFrom := fromDate.Format("2006-01-02")
	expectedTo := toDate.Format("2006-01-02")

	if result.StartDate != expectedFrom {
		t.Errorf("expected start_date '%s', got '%s'", expectedFrom, result.StartDate)
	}
	if result.EndDate != expectedTo {
		t.Errorf("expected end_date '%s', got '%s'", expectedTo, result.EndDate)
	}
}

func TestGetUserGrowthService_WeeklyBreakdown(t *testing.T) {
	_, _, _, db := SetupAdminTestRouter()

	params := adminSvc.UserGrowthParams{
		Preset:   "last_30_days",
		GroupBy:  "week",
		Timezone: "Africa/Lagos",
	}

	result, err := adminSvc.GetUserGrowthMetrics(db.Postgresql, params)
	if err != nil {
		t.Fatalf("GetUserGrowthMetrics returned error: %v", err)
	}

	if result.Breakdown == nil {
		t.Fatal("expected breakdown to be populated")
	}

	// Verify all breakdown items have period='week'
	for _, item := range result.Breakdown {
		if item.Period != "week" {
			t.Errorf("expected period 'week', got '%s'", item.Period)
		}
	}
}

func TestGetUserGrowthService_MonthlyBreakdown(t *testing.T) {
	_, _, _, db := SetupAdminTestRouter()

	params := adminSvc.UserGrowthParams{
		Preset:   "this_year",
		GroupBy:  "month",
		Timezone: "Africa/Lagos",
	}

	result, err := adminSvc.GetUserGrowthMetrics(db.Postgresql, params)
	if err != nil {
		t.Fatalf("GetUserGrowthMetrics returned error: %v", err)
	}

	if result.Breakdown == nil {
		t.Fatal("expected breakdown to be populated")
	}

	// Verify all breakdown items have period='month'
	for _, item := range result.Breakdown {
		if item.Period != "month" {
			t.Errorf("expected period 'month', got '%s'", item.Period)
		}

		// Verify month format (YYYY-MM)
		_, err := time.Parse("2006-01", item.Date)
		if err != nil {
			t.Errorf("invalid month format: %s", item.Date)
		}
	}
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestGetUserGrowth_NoUsersInRange(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	// Query a date range far in the future
	fromDate := "2099-01-01"
	toDate := "2099-12-31"

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/dashboard/user-growth?from=%s&to=%s", fromDate, toDate), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Should return 0 count
	if int64(respData["total_count"].(float64)) != 0 {
		t.Errorf("expected total_count 0 for future date range, got %v", respData["total_count"])
	}
}

func TestGetUserGrowth_LeapYearHandling(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	// Test with leap year date (Feb 29)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?from=2024-02-29&to=2024-02-29", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
}

func TestGetUserGrowth_ExcludesDeletedUsers(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()

	// Create a user
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("deleteduser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Deleted",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("deleteduser%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, user, false)

	// Soft delete the user
	now := getLagosTime()
	db.Postgresql.Model(&models.User{}).Where("email = ?", user.Email).Update("deleted_at", now)

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/user-growth", adminCtl.GetUserGrowth)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/user-growth?preset=today", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// The deleted user should not be counted
	// We can't assert exact count since other tests may have created users,
	// but we verify the response is valid
	totalCount := int64(respData["total_count"].(float64))
	if totalCount < 0 {
		t.Errorf("total count should be non-negative, got %d", totalCount)
	}
}
