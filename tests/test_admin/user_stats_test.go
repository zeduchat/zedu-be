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
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetFreeVsPaidUserStats(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := time.Now()

	userFree := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("statfree%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Free",
		LastName:    "Stat",
		Password:    "password",
		UserName:    fmt.Sprintf("statfree%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userFree, false)

	userPaid := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("statpaid%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Paid",
		LastName:    "Stat",
		Password:    "password",
		UserName:    fmt.Sprintf("statpaid%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userPaid, false)

	token := tst.GetLoginToken(t, gin.Default(), authCtl, models.LoginRequestModel{Email: userPaid.Email, Password: userPaid.Password})
	uid := tst.GetUserIDFromToken(t, token, db)

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:                 orgID,
		Name:               "Stat Paid Org",
		Email:              "statpaid@org.com",
		Type:               "business",
		Country:            "NG",
		OwnerID:            uid,
		SubscriptionPlanId: "pro",
		CreatedAt:          now,
	}
	if err := db.Postgresql.Create(&org).Error; err != nil {
		t.Fatalf("failed to create paid org: %v", err)
	}

	userOld := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("statold%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Old",
		LastName:    "Stat",
		Password:    "password",
		UserName:    fmt.Sprintf("statold%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userOld, false)

	eightDaysAgo := now.AddDate(0, 0, -8)
	db.Postgresql.Model(&models.User{}).Where("email = ?", userOld.Email).Update("created_at", eightDaysAgo)

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/users-stats", adminCtl.GetFreeVsPaidUserStats)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/users-stats?duration=last_week", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	chartData := respData["data"].([]any)

	if len(chartData) == 0 {
		t.Fatal("expected chart data, got empty")
	}

	foundExpectedData := false
	for _, pt := range chartData {
		point := pt.(map[string]any)
		freeCount := point["free"].(float64)
		paidCount := point["paid"].(float64)

		if freeCount >= 1 && paidCount >= 1 {
			foundExpectedData = true
			break
		}
	}

	if !foundExpectedData {
		t.Error("did not find data point with expected free and paid counts")
	}
}

func TestGetAICreditUsageStats(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := time.Now()

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:                 orgID,
		Name:               "AI Stat Org",
		Email:              "ai@org.com",
		Type:               "business",
		Country:            "US",
		SubscriptionPlanId: "pro",
		CreatedAt:          now,
	}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	userAI := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("aiuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "AI",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("aiuser%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userAI, false)

	token := tst.GetLoginToken(t, gin.Default(), authCtl, models.LoginRequestModel{Email: userAI.Email, Password: userAI.Password})
	uid := tst.GetUserIDFromToken(t, token, db)
	org.OwnerID = uid

	if err := db.Postgresql.Create(&org).Error; err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	agentID := utility.GenerateUUID()

	usage1 := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         100.0,
		AgentID:        agentID,
		CreatedAt:      now,
	}
	if err := db.Postgresql.Create(&usage1).Error; err != nil {
		t.Fatalf("failed to create usage1: %v", err)
	}

	usage2 := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         50.0,
		AgentID:        agentID,
		CreatedAt:      now.Add(-24 * time.Hour),
	}
	if err := db.Postgresql.Create(&usage2).Error; err != nil {
		t.Fatalf("failed to create usage2: %v", err)
	}

	usageOld := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         500.0,
		AgentID:        agentID,
		CreatedAt:      now.AddDate(0, 0, -8),
	}
	if err := db.Postgresql.Create(&usageOld).Error; err != nil {
		t.Fatalf("failed to create usageOld: %v", err)
	}
	db.Postgresql.Model(&models.CreditUsage{}).Where("id = ?", usageOld.ID).Update("created_at", usageOld.CreatedAt)

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/ai-credits", adminCtl.GetAICreditUsageStats)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/ai-credits?duration=weekly&unit=tokens", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	totalConsumed := respData["total_consumed"].(float64)

	if totalConsumed < 150 {
		t.Errorf("expected total consumed >= 150 (sum of usage1 and usage2), got %v", totalConsumed)
	}

	chartData := respData["data"].([]any)
	if len(chartData) == 0 {
		t.Fatal("expected chart data")
	}
}

func TestGetDashboardOverviewStats(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	currUUID := utility.GenerateUUID()
	now := time.Now()

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:                 orgID,
		Name:               "Overview Org",
		Email:              "overview@org.com",
		Type:               "business",
		Country:            "US",
		OwnerID:            currUUID,
		SubscriptionPlanId: "pro",
		CreatedAt:          now,
	}

	userOwner := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("owner%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Owner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("owner%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userOwner, false)
	token := tst.GetLoginToken(t, gin.Default(), authCtl, models.LoginRequestModel{Email: userOwner.Email, Password: userOwner.Password})
	uid := tst.GetUserIDFromToken(t, token, db)
	org.OwnerID = uid
	db.Postgresql.Create(&org)

	userLastMonth := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("lastmonth%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Last",
		LastName:    "Month",
		Password:    "password",
		UserName:    fmt.Sprintf("lastmonth%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userLastMonth, false)
	db.Postgresql.Model(&models.User{}).Where("email = ?", userLastMonth.Email).Update("created_at", now.AddDate(0, -1, 0))

	topup1 := models.CreditTransaction{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Type:           "Top-up",
		CreatedAt:      now,
		Amount:         250.0,
	}
	db.Postgresql.Create(&topup1)

	topup2 := models.CreditTransaction{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         250.0,
		Type:           "Top-up",
		CreatedAt:      now,
	}
	db.Postgresql.Create(&topup2)

	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/overview", adminCtl.GetDashboardOverviewStats)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/overview", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	revenue := respData["revenue"].(map[string]any)
	if revenue["value"].(float64) < 500 {
		t.Errorf("Expected current revenue >= 500, got %v", revenue["value"])
	}

	newUsers := respData["new_users"].(map[string]any)
	if newUsers["value"].(float64) < 1 {
		t.Errorf("Expected at least 1 new user (owner), got %v", newUsers["value"])
	}

	totalUsers := respData["total_users"].(map[string]any)
	if totalUsers["value"].(float64) < 2 {
		t.Errorf("Expected at least 2 users (owner + lastmonth), got %v", totalUsers["value"])
	}

	if _, ok := respData["available_credits"]; !ok {
		t.Error("expected available_credits field")
	}

}
