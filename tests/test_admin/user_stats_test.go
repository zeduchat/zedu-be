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

	// 1. Create Free User (Today)
	userFree := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("statfree%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Free",
		LastName:    "Stat",
		Password:    "password",
		UserName:    fmt.Sprintf("statfree%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userFree, false)

	// 2. Create Paid User (Today)
	userPaid := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("statpaid%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Paid",
		LastName:    "Stat",
		Password:    "password",
		UserName:    fmt.Sprintf("statpaid%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authCtl, userPaid, false)

	// Make userPaid actually paid
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

	// 3. Create Old User (8 days ago - outside of last_week)
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

	// Setup Router
	r := gin.Default()
	r.GET("/api/v1/backoffice/dashboard/users-stats", adminCtl.GetFreeVsPaidUserStats)

	// Execute Request
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/dashboard/users-stats?duration=last_week", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Assertions
	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	chartData := respData["data"].([]any)

	if len(chartData) == 0 {
		t.Fatal("expected chart data, got empty")
	}

	// Verify today's data
	todayStr := now.Format("2006-01-02")
	foundToday := false

	for _, pt := range chartData {
		point := pt.(map[string]any)
		if point["date"] == todayStr {
			foundToday = true

			freeCount := point["free"].(float64)
			paidCount := point["paid"].(float64)

			// We expect AT LEAST 1 Free and 1 Paid from our seeding
			// (Other concurrent tests might add more, so exact match is flaky)
			if freeCount < 1 {
				t.Errorf("expected at least 1 free user today, got %v", freeCount)
			}
			if paidCount < 1 {
				t.Errorf("expected at least 1 paid user today, got %v", paidCount)
			}
		}
	}

	if !foundToday {
		t.Errorf("did not find data point for today: %s", todayStr)
	}
}
