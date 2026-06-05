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

func TestGetInvitationDashboard_Success(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)

	// Verify structure contains all expected fields
	if _, ok := respData["stats"]; !ok {
		t.Error("missing 'stats' in response")
	}
	if _, ok := respData["growth_trends"]; !ok {
		t.Error("missing 'growth_trends' in response")
	}
	if _, ok := respData["top_inviters"]; !ok {
		t.Error("missing 'top_inviters' in response")
	}
	if _, ok := respData["invitations"]; !ok {
		t.Error("missing 'invitations' in response")
	}
}

func TestGetInvitationDashboard_IncludeStatsTrue(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard?include_stats=true", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["stats"].(map[string]any)

	// Verify stats has expected keys
	if _, ok := stats["total_invitations_sent"]; !ok {
		t.Error("missing total_invitations_sent in stats")
	}
	if _, ok := stats["sent_today"]; !ok {
		t.Error("missing sent_today in stats")
	}
	if _, ok := stats["yesterday"]; !ok {
		t.Error("missing yesterday in stats")
	}
	if _, ok := stats["this_week"]; !ok {
		t.Error("missing this_week in stats")
	}
}

func TestGetInvitationDashboard_IncludeStatsFalse(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard?include_stats=false", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["stats"].(map[string]any)

	// When include_stats=false, stats should be zero
	totalSent := stats["total_invitations_sent"].(float64)
	if totalSent != 0 {
		t.Errorf("expected total_invitations_sent to be 0 when include_stats=false, got %v", totalSent)
	}
}

func TestGetInvitationDashboard_GrowthTrends(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	growthTrends := respData["growth_trends"].([]any)

	// Should have 6 months of data
	if len(growthTrends) != 6 {
		t.Errorf("expected 6 growth trend points, got %d", len(growthTrends))
	}

	// Each point should have month, organic, referral
	for _, point := range growthTrends {
		p := point.(map[string]any)
		if _, ok := p["month"]; !ok {
			t.Error("missing 'month' in growth trend point")
		}
		if _, ok := p["organic"]; !ok {
			t.Error("missing 'organic' in growth trend point")
		}
		if _, ok := p["referral"]; !ok {
			t.Error("missing 'referral' in growth trend point")
		}
	}
}

func TestGetInvitationDashboard_TopInviters(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard?top_limit=3", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	topInviters := respData["top_inviters"].([]any)

	// Should not exceed top_limit
	if len(topInviters) > 3 {
		t.Errorf("expected max 3 top inviters, got %d", len(topInviters))
	}

	// Verify structure if any exist
	for _, inviter := range topInviters {
		inv := inviter.(map[string]any)
		if _, ok := inv["id"]; !ok {
			t.Error("missing 'id' in top inviter")
		}
		if _, ok := inv["name"]; !ok {
			t.Error("missing 'name' in top inviter")
		}
		if _, ok := inv["invite_count"]; !ok {
			t.Error("missing 'invite_count' in top inviter")
		}
		if _, ok := inv["rank"]; !ok {
			t.Error("missing 'rank' in top inviter")
		}
	}
}

func TestGetInvitationDashboard_FilterByStatus(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	// Filter by pending status
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard?status=invited", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
}

func TestGetInvitationDashboard_SearchByEmail(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard?search=test@example.com", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
}

func TestGetInvitationDashboard_Pagination(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard?page=1&limit=5", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	invitations := respData["invitations"].([]any)

	// Should not exceed limit
	if len(invitations) > 5 {
		t.Errorf("expected max 5 invitations, got %d", len(invitations))
	}
}

func TestGetInvitationDashboardService_Stats(t *testing.T) {
	_, _, _, db := SetupAdminTestRouter()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil)
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	filter := adminSvc.InvitationFilter{
		IncludeStats: true,
		TopLimit:     5,
	}

	response, _, code, err := adminSvc.GetInvitationDashboard(db.Postgresql, c, filter)
	if err != nil {
		t.Fatalf("GetInvitationDashboard service returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}

	// Verify stats is populated
	if response.Stats.TotalInvitationsSent < 0 {
		t.Error("total_invitations_sent should be >= 0")
	}
}

func TestGetInvitationDashboardService_WithInvitation(t *testing.T) {
	_, authCtl, _, db := SetupAdminTestRouter()

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       "invite-dash-" + currUUID + "@qa.team",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "InviteDash",
		LastName:    "User",
		Password:    "password",
		UserName:    "invite-dash-" + currUUID,
	}

	tst.SignupUser(t, gin.Default(), *authCtl, user1, false)

	token := tst.GetLoginToken(t, gin.Default(), *authCtl, models.LoginRequestModel{Email: user1.Email, Password: user1.Password})
	uid := tst.GetUserIDFromToken(t, token, db)

	// Create an invitation
	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:      orgID,
		Name:    "Invite Dash Org",
		Email:   "invite@org.com",
		Type:    "business",
		Country: "NG",
		OwnerID: uid,
	}
	db.Postgresql.Create(&org)

	invitation := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          "invited@example.com",
		Status:         "invited",
		OrganisationID: orgID,
		InvitedBy:      uid,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Role:           utility.GenerateUUID(),
	}
	db.Postgresql.Create(&invitation)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil)
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	filter := adminSvc.InvitationFilter{
		IncludeStats: true,
		TopLimit:     5,
	}

	response, _, code, err := adminSvc.GetInvitationDashboard(db.Postgresql, c, filter)
	if err != nil {
		t.Fatalf("GetInvitationDashboard service returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}

	if len(response.Invitations) < 1 {
		t.Log("Warning: expected at least 1 invitation in response")
	}

	if response.Stats.TotalInvitationsSent < 1 {
		t.Log("Warning: expected total_invitations_sent >= 1")
	}

	// Cleanup
	db.Postgresql.Delete(&invitation)
	db.Postgresql.Delete(&org)
}

func TestGetInvitationDashboard_InvitesConversionPresent(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	r.GET("/api/v1/backoffice/admins/invitations/dashboard", adminCtl.GetInvitationDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/invitations/dashboard?include_stats=true", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	respData := data["data"].(map[string]any)
	stats := respData["stats"].(map[string]any)

	// Verify invites_conversion field exists
	if _, ok := stats["invites_conversion"]; !ok {
		t.Error("missing 'invites_conversion' in stats")
	}

	// Verify it's a valid number (>=0)
	conversionVal := stats["invites_conversion"].(float64)
	if conversionVal < 0 {
		t.Errorf("expected invites_conversion to be >= 0, got %v", conversionVal)
	}
}

func TestGetInvitationDashboardService_ConversionCalculation(t *testing.T) {
	_, authCtl, _, db := SetupAdminTestRouter()

	tx := db.Postgresql.Begin()
	defer tx.Rollback()

	// Clear existing invitations to ensure isolated test
	tx.Exec("DELETE FROM invitations")

	authCtlTx := auth.Controller{
		Db:        &storage.Database{Postgresql: tx},
		Validator: authCtl.Validator,
		Logger:    authCtl.Logger,
		ExtReq:    authCtl.ExtReq,
	}

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       "conversion-test-" + currUUID + "@qa.team",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "ConversionTest",
		LastName:    "User",
		Password:    "password",
		UserName:    "conversion-test-" + currUUID,
	}

	tst.SignupUser(t, gin.Default(), authCtlTx, user1, false)

	token := tst.GetLoginToken(t, gin.Default(), authCtlTx, models.LoginRequestModel{Email: user1.Email, Password: user1.Password})
	uid := tst.GetUserIDFromToken(t, token, db)

	// Create test organization
	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:      orgID,
		Name:    "Conversion Test Org",
		Email:   "conversion-test@org.com",
		Type:    "business",
		Country: "NG",
		OwnerID: uid,
	}
	tx.Create(&org)

	roleID := utility.GenerateUUID()
	now := time.Now()
	last30Days := now.AddDate(0, 0, -30)
	last60Days := now.AddDate(0, 0, -60)

	var createdInvitations []models.Invitation

	// Create 10 invitations in last 30 days: 5 "accepted", 5 "invited" = 50% conversion
	for i := 0; i < 10; i++ {
		status := "invited"
		if i < 5 {
			status = "accepted"
		}

		invitation := models.Invitation{
			ID:             utility.GenerateUUID(),
			Email:          fmt.Sprintf("user%d-last30@example.com", i),
			Status:         status,
			OrganisationID: orgID,
			InvitedBy:      uid,
			CreatedAt:      last30Days.AddDate(0, 0, 5),
			ExpiresAt:      last30Days.AddDate(0, 0, 5).Add(24 * time.Hour),
			Role:           roleID,
		}
		tx.Create(&invitation)
		createdInvitations = append(createdInvitations, invitation)
	}

	// Create 5 invitations from 30-60 days ago: 2 "accepted", 3 "invited" = 40% conversion
	for i := 0; i < 5; i++ {
		status := "invited"
		if i < 2 {
			status = "accepted"
		}

		invitation := models.Invitation{
			ID:             utility.GenerateUUID(),
			Email:          fmt.Sprintf("user%d-30to60@example.com", i),
			Status:         status,
			OrganisationID: orgID,
			InvitedBy:      uid,
			CreatedAt:      last60Days.AddDate(0, 0, 5),
			ExpiresAt:      last60Days.AddDate(0, 0, 5).Add(24 * time.Hour),
			Role:           roleID,
		}
		tx.Create(&invitation)
		createdInvitations = append(createdInvitations, invitation)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil)
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	filter := adminSvc.InvitationFilter{
		IncludeStats: true,
		TopLimit:     5,
	}

	response, _, code, err := adminSvc.GetInvitationDashboard(tx, c, filter)
	if err != nil {
		t.Fatalf("GetInvitationDashboard service returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}

	// Verify conversion calculation for last 30 days only: 5 accepted / 10 total = 50%
	expectedConversion := 50.0
	actualConversion := response.Stats.InvitesConversionMonth

	// Allow small floating point difference
	diff := actualConversion - expectedConversion
	if diff < -0.01 || diff > 0.01 {
		t.Errorf("expected invites_conversion to be ~%.2f (last 30 days only), got %.2f", expectedConversion, actualConversion)
	}

	// Verify conversion change percentage: (50 - 40) / 40 * 100 = 25%
	expectedChangePercent := 25.0
	actualChangePercent := response.Stats.InvitesConversionChangePercent

	changeDiff := actualChangePercent - expectedChangePercent
	if changeDiff < -0.01 || changeDiff > 0.01 {
		t.Errorf("expected invites_conversion_change_percent to be ~%.2f, got %.2f", expectedChangePercent, actualChangePercent)
	}

	// Cleanup
	for _, inv := range createdInvitations {
		tx.Delete(&inv)
	}
	tx.Delete(&org)
}
