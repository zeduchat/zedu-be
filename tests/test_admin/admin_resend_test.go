package test_admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/invitation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func setupAdminResendRouter(t *testing.T) (*gin.Engine, *auth.Controller, *storage.Database, func(ext, token string) *httptest.ResponseRecorder) {
	t.Helper()
	r, authCtl, logger, db := SetupAdminTestRouter()

	inviteCtl := invitation.Controller{Db: db, Validator: authCtl.Validator, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r.POST("/api/v1/invite/admin-resend", middleware.AdminAuthorize(db.Postgresql), middleware.RequireSuperAdmin(), inviteCtl.AdminResend)

	send := func(ext, token string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"extension": ext, "time_from": "2000-01-01"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/invite/admin-resend", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Referer", "https://evil.test/")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	return r, authCtl, db, send
}

func TestAdminResend_RequiresSuperAdmin(t *testing.T) {
	r, authCtl, db, send := setupAdminResendRouter(t)

	// anonymous
	tst.AssertStatusCode(t, send("@acme.test", "").Code, http.StatusUnauthorized)

	// normal user JWT has no admin_id claim
	uid := utility.RandomString(8)
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("resenduser%v@qa.team", uid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "resend",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("resenduser%v", uid),
	}
	tst.SignupUser(t, r, *authCtl, user, false)
	userToken := tst.GetLoginToken(t, r, *authCtl, models.LoginRequestModel{Email: user.Email, Password: user.Password})
	tst.AssertStatusCode(t, send("@acme.test", userToken).Code, http.StatusUnauthorized)

	// plain admin, not superadmin
	adminToken := CreateAdminAndGetToken(t, r, db, models.RoleAdmin)
	tst.AssertStatusCode(t, send("@acme.test", adminToken).Code, http.StatusForbidden)
}

func TestAdminResend_SuperAdminValidationAndScope(t *testing.T) {
	r, _, db, send := setupAdminResendRouter(t)
	token := CreateSuperAdminAndGetToken(t, r, db)

	// wildcard / non-domain extensions are rejected
	for _, ext := range []string{"%", "@", "@%", "@a_b.test", "acme.test"} {
		rr := send(ext, token)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("extension %q: expected 400, got %d (%s)", ext, rr.Code, rr.Body.String())
		}
	}

	// no matches must not panic
	tst.AssertStatusCode(t, send("@nomatch-"+utility.RandomString(6)+".test", token).Code, http.StatusOK)

	// matched invites in two orgs; an invite to the same address that is
	// outside the date window must not have its expiry extended.
	domain := "@resend-" + utility.RandomString(6) + ".test"
	orgA := CreateOrganizationWithCredit(t, db.Postgresql, 0)
	orgB := CreateOrganizationWithCredit(t, db.Postgresql, 0)
	old := time.Now().Add(-time.Hour).UTC()
	mk := func(org, email string) models.Invitation {
		inv := models.Invitation{
			ID:             utility.GenerateUUID(),
			Email:          email,
			Token:          utility.GenerateUUID(),
			Status:         "invited",
			Role:           utility.GenerateUUID(),
			OrganisationID: org,
			InvitedBy:      utility.GenerateUUID(),
			ExpiresAt:      old,
		}
		if err := db.Postgresql.Create(&inv).Error; err != nil {
			t.Fatalf("create invitation: %v", err)
		}
		return inv
	}
	a := mk(orgA, "x"+domain)
	b := mk(orgB, "y"+domain)
	other := mk(orgA, "x@other-"+utility.RandomString(6)+".test")
	t.Cleanup(func() {
		db.Postgresql.Where("id IN ?", []string{a.ID, b.ID, other.ID}).Delete(&models.Invitation{})
	})

	tst.AssertStatusCode(t, send(domain, token).Code, http.StatusOK)

	for _, inv := range []models.Invitation{a, b, other} {
		var got models.Invitation
		if err := db.Postgresql.First(&got, "id = ?", inv.ID).Error; err != nil {
			t.Fatalf("reload invitation: %v", err)
		}
		extended := got.ExpiresAt.After(time.Now().Add(24 * time.Hour))
		if inv.ID == other.ID && extended {
			t.Errorf("invite %s outside the extension was extended", inv.ID)
		}
		if inv.ID != other.ID && !extended {
			t.Errorf("matched invite %s was not extended (expires_at %v)", inv.ID, got.ExpiresAt)
		}
	}
}
