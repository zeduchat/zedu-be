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

func TestListUsersEndpoint(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("listuser1%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user1",
		Password:    "password",
		UserName:    fmt.Sprintf("listuser1%v", currUUID),
	}
	user2 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("listuser2%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user2",
		Password:    "password",
		UserName:    fmt.Sprintf("listuser2%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)
	tst.SignupUser(t, gin.Default(), authCtl, user2, false)

	r.GET("/api/v1/backoffice/admins/users", adminCtl.ListUsers)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users?page=1&limit=10", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	rr2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rr2)
	c2.Request = req
	adminCtl.ListUsers(c2)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	var bodyRecorder *httptest.ResponseRecorder
	if rr2.Body.Len() > 0 {
		bodyRecorder = rr2
	} else {
		bodyRecorder = rr
	}

	data := tst.ParseResponse(bodyRecorder)
	code := 0
	if data["status_code"] != nil {
		code = int(data["status_code"].(float64))
	}
	tst.AssertStatusCode(t, code, http.StatusOK)
	tst.AssertResponseMessage(t, data["message"].(string), "users retrieved successfully")

	respData := data["data"].(map[string]any)
	arr := respData["users"].([]any)
	if len(arr) < 2 {
		t.Fatalf("expected at least 2 users, got %d", len(arr))
	}

	for i, item := range arr {
		m := item.(map[string]any)
		if _, ok := m["subscription_status"]; !ok {
			t.Errorf("user %d missing 'subscription_status' field", i)
		}
		if _, ok := m["credit_used"]; !ok {
			t.Errorf("user %d missing 'credit_used' field", i)
		}
		if _, ok := m["amount_spent"]; !ok {
			t.Errorf("user %d missing 'amount_spent' field", i)
		}
	}

	// Verify Stats keys exist
	stats := respData["stats"].(map[string]any)
	if _, ok := stats["total_signups"]; !ok {
		t.Error("missing total_signups in stats")
	}

	pagArr, ok := data["pagination"].([]any)
	if !ok || len(pagArr) == 0 {
		t.Logf("response headers: %v", rr.Header())
		t.Logf("response body: %s", rr.Body.String())
		t.Fatalf("expected pagination metadata in response")
	}
	if _, ok := pagArr[0].(map[string]any); !ok {
		t.Fatalf("expected pagination element to be an object")
	}
}

func TestInviteLeaderboardEndpoint(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	adminCtl := admin.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	authCtl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("inviteuser1%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "inv",
		LastName:    "user1",
		Password:    "password",
		UserName:    fmt.Sprintf("inviteuser1%v", currUUID),
	}
	user2 := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("inviteuser2%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "inv",
		LastName:    "user2",
		Password:    "password",
		UserName:    fmt.Sprintf("inviteuser2%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authCtl, user1, false)
	tst.SignupUser(t, gin.Default(), authCtl, user2, false)

	token1 := tst.GetLoginToken(t, gin.Default(), authCtl, models.LoginRequestModel{Email: user1.Email, Password: user1.Password})
	if token1 == "" {
		t.Fatal("failed to get token for user1")
	}
	uid1 := tst.GetUserIDFromToken(t, token1, db)

	token2 := tst.GetLoginToken(t, gin.Default(), authCtl, models.LoginRequestModel{Email: user2.Email, Password: user2.Password})
	if token2 == "" {
		t.Fatal("failed to get token for user2")
	}
	uid2 := tst.GetUserIDFromToken(t, token2, db)

	orgID := utility.GenerateUUID()
	now := time.Now().UTC()

	invites := []models.Invitation{
		{ID: utility.GenerateUUID(), Email: "a1@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid1, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
		{ID: utility.GenerateUUID(), Email: "a2@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid1, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
		{ID: utility.GenerateUUID(), Email: "a3@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid1, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
		{ID: utility.GenerateUUID(), Email: "b1@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid2, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
	}

	org := models.Organisation{ID: orgID, Name: "Test Org", Email: "org@example.com", Type: "test", Country: "NG", OwnerID: utility.GenerateUUID(), CreatedAt: now}
	if err := db.Postgresql.Create(&org).Error; err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	tErr := db.Postgresql.Create(&invites).Error
	if tErr != nil {
		t.Fatalf("failed to create invites: %v", tErr)
	}

	r.GET("/api/v1/backoffice/admins/users/invites", adminCtl.InviteLeaderboard)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/backoffice/admins/users/invites?limit=10", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	data := tst.ParseResponse(rr)
	code := int(data["status_code"].(float64))
	tst.AssertStatusCode(t, code, http.StatusOK)
	tst.AssertResponseMessage(t, data["message"].(string), "invite leaderboard retrieved successfully")

	arr := data["data"].([]any)
	if len(arr) < 1 {
		t.Fatalf("expected at least 1 leaderboard entry, got %d", len(arr))
	}

	for i, item := range arr {
		m := item.(map[string]any)
		if _, ok := m["id"]; !ok {
			t.Fatalf("entry %d missing 'id' field", i)
		}
		if _, ok := m["referrals"]; !ok {
			t.Fatalf("entry %d missing 'referrals' field", i)
		}
		ref := int(m["referrals"].(float64))
		if ref < 1 {
			t.Logf("entry %d has referrals=%d", i, ref)
		}
	}

	hasHighReferrals := false
	for _, item := range arr {
		m := item.(map[string]any)
		ref := int(m["referrals"].(float64))
		if ref >= 3 {
			hasHighReferrals = true
			break
		}
	}
	if !hasHighReferrals {
		t.Fatalf("expected at least one user with 3+ referrals in leaderboard")
	}
}

func TestListUsersService(t *testing.T) {
	_, authCtl, _, db := SetupAdminTestRouter()

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       "svc-list1-" + currUUID + "@qa.team",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "svc",
		LastName:    "user1",
		Password:    "password",
		UserName:    "svc-list1-" + currUUID,
	}
	user2 := models.CreateUserRequestModel{
		Email:       "svc-list2-" + currUUID + "@qa.team",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "svc",
		LastName:    "user2",
		Password:    "password",
		UserName:    "svc-list2-" + currUUID,
	}

	tst.SignupUser(t, gin.Default(), *authCtl, user1, false)
	tst.SignupUser(t, gin.Default(), *authCtl, user2, false)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil)
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	response, pagination, code, err := adminSvc.ListUsers(db.Postgresql, c, adminSvc.UserFilter{})
	if err != nil {
		t.Fatalf("ListUsers service returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}
	if len(response.Users) < 2 {
		t.Fatalf("expected at least 2 users, got %d", len(response.Users))
	}
	if pagination.TotalItems < 2 {
		t.Fatalf("expected pagination TotalItems >= 2, got %d", pagination.TotalItems)
	}

}

func TestListUsersByInvitesService(t *testing.T) {
	_, authCtl, _, db := SetupAdminTestRouter()

	currUUID := utility.GenerateUUID()
	user1 := models.CreateUserRequestModel{
		Email:       "svc-invite1-" + currUUID + "@qa.team",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "inv",
		LastName:    "user1",
		Password:    "password",
		UserName:    "svc-invite1-" + currUUID,
	}
	user2 := models.CreateUserRequestModel{
		Email:       "svc-invite2-" + currUUID + "@qa.team",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "inv",
		LastName:    "user2",
		Password:    "password",
		UserName:    "svc-invite2-" + currUUID,
	}

	tst.SignupUser(t, gin.Default(), *authCtl, user1, false)
	tst.SignupUser(t, gin.Default(), *authCtl, user2, false)

	token1 := tst.GetLoginToken(t, gin.Default(), *authCtl, models.LoginRequestModel{Email: user1.Email, Password: user1.Password})
	if token1 == "" {
		t.Fatal("failed to get token for user1")
	}
	uid1 := tst.GetUserIDFromToken(t, token1, db)

	token2 := tst.GetLoginToken(t, gin.Default(), *authCtl, models.LoginRequestModel{Email: user2.Email, Password: user2.Password})
	if token2 == "" {
		t.Fatal("failed to get token for user2")
	}
	uid2 := tst.GetUserIDFromToken(t, token2, db)

	orgID := utility.GenerateUUID()
	now := time.Now().UTC()

	org := models.Organisation{ID: orgID, Name: "Test Org", Email: "org@example.com", Type: "test", Country: "NG", OwnerID: utility.GenerateUUID(), CreatedAt: now}
	if err := db.Postgresql.Create(&org).Error; err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	invites := []models.Invitation{
		{ID: utility.GenerateUUID(), Email: "a1@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid1, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
		{ID: utility.GenerateUUID(), Email: "a2@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid1, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
		{ID: utility.GenerateUUID(), Email: "a3@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid1, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
		{ID: utility.GenerateUUID(), Email: "b1@example.com", Token: utility.GenerateUUID(), Status: "invited", Role: utility.GenerateUUID(), OrganisationID: orgID, InvitedBy: uid2, IsTelexUser: false, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)},
	}

	tErr := db.Postgresql.Create(&invites).Error
	if tErr != nil {
		t.Fatalf("failed to create invites: %v", tErr)
	}

	users, code, err := adminSvc.ListUsersByInvites(db.Postgresql, &orgID, 10)
	if err != nil {
		t.Fatalf("ListUsersByInvites returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}
	if len(users) < 2 {
		t.Fatalf("expected at least 2 leaderboard entries, got %d", len(users))
	}

	top := users[0]
	if int(top["referrals"].(int)) != 3 {
		t.Fatalf("expected top referrals 3, got %v", top["referrals"])
	}

	second := users[1]
	if int(second["referrals"].(int)) != 1 {
		t.Fatalf("expected second referrals 1, got %v", second["referrals"])
	}

	_, code2, err2 := adminSvc.ListUsersByInvites(db.Postgresql, nil, 101)
	if err2 == nil || code2 != http.StatusBadRequest {
		t.Fatalf("expected bad request for limit > 100, got code %d err %v", code2, err2)
	}
}
