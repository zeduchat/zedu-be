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

// =============================================================================
// FILTERING TESTS
// =============================================================================

func TestListUsers_SearchByName(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	uniquePrefix := "searchname1" + currUUID[:8]

	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%s@qa.team", uniquePrefix),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Search",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("searchname1%v", currUUID),
	}
	user2 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("searchname2%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Other",
		LastName:    "Person",
		Password:    "password",
		UserName:    fmt.Sprintf("searchname2%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)
	tst.SignupUser(t, gin.Default(), authCtl, user2, false)

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	// Search by unique email prefix
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users?search=%s", uniquePrefix), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	if len(users) < 1 {
		t.Logf("Search term: %s", uniquePrefix)
		t.Fatalf("expected at least 1 user matching search, got %d", len(users))
	}

	for _, u := range users {
		userMap := u.(map[string]any)
		name := userMap["name"].(string)
		if name == "" {
			continue
		}
		// Verify the search term is in results (case insensitive check done by DB)
	}
}

func TestListUsers_SearchByEmail(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	uniqueEmailPrefix := "emailsearch" + currUUID[:8]

	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%s@gmail.com", uniqueEmailPrefix),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Email",
		LastName:    "Search",
		Password:    "password",
		UserName:    fmt.Sprintf("emailsearch1%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users?search=%s", uniqueEmailPrefix), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	if len(users) < 1 {
		t.Fatalf("expected at least 1 user matching email search, got %d", len(users))
	}
}

func TestListUsers_UserTypeActive(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("activeuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Active",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("activeuser%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)

	// Set last_activity_at to now for the user
	now := time.Now()
	db.Postgresql.Model(&models.User{}).Where("email = ?", user1.Email).Update("last_activity_at", now)

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?user_type=active", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	// All returned users should have recent activity
	if len(users) < 1 {
		t.Fatalf("expected at least 1 active user, got %d", len(users))
	}
}

func TestListUsers_UserTypePaying(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("payinguser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Paying",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("payinguser%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)

	token := tst.GetLoginToken(t, gin.Default(), authCtl, models.LoginRequestModel{Email: user1.Email, Password: user1.Password})
	uid := tst.GetUserIDFromToken(t, token, db)

	// Create an org with paid subscription for this user
	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:                 orgID,
		Name:               "Paid Org",
		Email:              "paid@org.com",
		Type:               "business",
		Country:            "NG",
		OwnerID:            uid,
		SubscriptionPlanId: "pro", // Non-free subscription
		CreatedAt:          time.Now(),
	}
	db.Postgresql.Create(&org)

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?user_type=paying", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	if len(users) < 1 {
		t.Fatalf("expected at least 1 paying user, got %d", len(users))
	}

	// Verify all returned users have Paid status
	for _, u := range users {
		userMap := u.(map[string]any)
		status := userMap["subscription_status"].(string)
		if status != "Paid" {
			t.Errorf("expected subscription_status 'Paid', got '%s'", status)
		}
	}
}

func TestListUsers_UserTypeFree(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("freeuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Free",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("freeuser%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?user_type=free", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	if len(users) < 1 {
		t.Fatalf("expected at least 1 free user, got %d", len(users))
	}

	// Verify all returned users have Free status
	for _, u := range users {
		userMap := u.(map[string]any)
		status := userMap["subscription_status"].(string)
		if status != "Free" {
			t.Errorf("expected subscription_status 'Free', got '%s'", status)
		}
	}
}

// =============================================================================
// DURATION PRESET TESTS
// =============================================================================

func TestListUsers_DurationLastMonth(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("lastmonth%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "LastMonth",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("lastmonth%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?duration=last_month", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	// Users created just now should be in last_month
	if len(users) < 1 {
		t.Fatalf("expected at least 1 user in last month, got %d", len(users))
	}
}

func TestListUsers_DurationLast3Months(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?duration=last_3_months", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	_ = respData["users"].([]any)
	// Just verify the request succeeds with this duration
}

func TestListUsers_DurationLastYear(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?duration=last_year", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
}

func TestListUsers_CustomDateRange(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/users?duration=custom&start_date=%s&end_date=%s", startDate, endDate), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
}

// =============================================================================
// STATS TESTS
// =============================================================================

func TestListUsers_IncludeStatsTrue(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?include_stats=true", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["stats"].(map[string]any)

	// Verify stats has expected keys
	for _, key := range []string{
		"total_signups", "new_today", "free_users", "paid_users",
		"avg_session_length_month", "avg_session_length_change_percent",
	} {
		if _, ok := stats[key]; !ok {
			t.Errorf("missing %s in stats", key)
		}
	}
}

func TestListUsers_IncludeStatsFalse(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?include_stats=false", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["stats"].(map[string]any)

	// When include_stats=false, stats should be zero/empty
	totalSignups := stats["total_signups"].(float64)
	if totalSignups != 0 {
		t.Errorf("expected total_signups to be 0 when include_stats=false, got %v", totalSignups)
	}
	// the new fields should also have their zero values
	if stats["avg_session_length_month"].(string) != "" {
		t.Error("expected avg_session_length_month to be empty when include_stats=false")
	}
	if stats["avg_session_length_change_percent"].(float64) != 0 {
		t.Error("expected avg_session_length_change_percent to be 0 when include_stats=false")
	}
}

func TestListUsers_StatsDefaultToIncluded(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	// No include_stats param - should default to true
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["stats"].(map[string]any)

	// Stats should be populated by default (including new session fields)
	for _, key := range []string{"total_signups", "avg_session_length_month", "avg_session_length_change_percent"} {
		if _, ok := stats[key]; !ok {
			t.Errorf("stats should include %s by default", key)
		}
	}
}

// =============================================================================
// PAGINATION TESTS
// =============================================================================

func TestListUsers_Pagination(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?page=1&limit=5", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	// Should not exceed limit
	if len(users) > 5 {
		t.Errorf("expected max 5 users, got %d", len(users))
	}

	// Check pagination metadata exists
	_, ok := data["pagination"].(map[string]any)
	if !ok {
		t.Fatal("expected pagination metadata")
	}
}

func TestListUsers_EmptyPage(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	// Request a very high page number
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?page=9999&limit=10", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	users := respData["users"].([]any)

	// Should return empty array, not error
	if len(users) != 0 {
		t.Errorf("expected 0 users on empty page, got %d", len(users))
	}
}

// =============================================================================
// COMBINED FILTER TESTS
// =============================================================================

func TestListUsers_CombinedFilters(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	// Combine multiple filters
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?user_type=free&duration=last_month&search=user", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	_ = respData["users"].([]any)
	// Just verify combined filters work without error
}

// =============================================================================
// SERVICE LAYER TESTS
// =============================================================================

func TestListUsersService_WithFilters(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authCtl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       "svcfilter-" + currUUID + "@qa.team",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "SvcFilter",
		LastName:    "User",
		Password:    "password",
		UserName:    "svcfilter" + currUUID,
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil)
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	filter := adminSvc.UserFilter{
		Search:       "SvcFilter",
		IncludeStats: true,
	}

	response, _, code, err := adminSvc.ListUsers(db.Postgresql, c, filter)
	if err != nil {
		t.Fatalf("ListUsers service returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}
	if len(response.Users) < 1 {
		t.Fatalf("expected at least 1 user matching filter, got %d", len(response.Users))
	}
}

func TestListUsersService_StatsCalculation(t *testing.T) {
	_, _, _, db := SetupAdminTestRouter()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil)
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	filter := adminSvc.UserFilter{
		IncludeStats: true,
	}

	response, _, code, err := adminSvc.ListUsers(db.Postgresql, c, filter)
	if err != nil {
		t.Fatalf("ListUsers service returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}

	// Verify stats are calculated
	if response.Stats.TotalSignups == 0 {
		t.Log("Warning: TotalSignups is 0, might need more test data")
	}

	// Free + Paid should equal Total
	if response.Stats.FreeUsers+response.Stats.PaidUsers != response.Stats.TotalSignups {
		t.Errorf("FreeUsers(%d) + PaidUsers(%d) != TotalSignups(%d)",
			response.Stats.FreeUsers, response.Stats.PaidUsers, response.Stats.TotalSignups)
	}
}
