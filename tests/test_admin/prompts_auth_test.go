package test_admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/router"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestPromptsRoutes_RequireBackofficeAdmin(t *testing.T) {
	r, authCtl, logger, db := SetupAdminTestRouter()
	router.Translator(r, "/api/v1", validator.New(), db, logger)

	name := "prompt-auth-test-" + utility.RandomString(8)
	t.Cleanup(func() { db.Postgresql.Where("name = ?", name).Delete(&models.Prompts{}) })

	maxVersion := func() int {
		var v int
		db.Postgresql.Model(&models.Prompts{}).Where("name = ?", name).Select("COALESCE(MAX(version), 0)").Scan(&v)
		return v
	}
	do := func(method, path, token string, body any) int {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req, _ := http.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}
	valid := map[string]string{"name": name, "template": "known-good template"}

	// anonymous
	tst.AssertStatusCode(t, do(http.MethodPost, "/api/v1/prompts/", "", valid), http.StatusUnauthorized)
	tst.AssertStatusCode(t, do(http.MethodGet, "/api/v1/prompts/", "", nil), http.StatusUnauthorized)
	tst.AssertStatusCode(t, do(http.MethodGet, "/api/v1/prompts/"+name, "", nil), http.StatusUnauthorized)
	tst.AssertStatusCode(t, do(http.MethodGet, "/api/v1/prompts/steps", "", nil), http.StatusUnauthorized)

	// normal user JWT has no admin_id claim
	uid := utility.RandomString(8)
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("promptuser%v@qa.team", uid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "prompt",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("promptuser%v", uid),
	}
	tst.SignupUser(t, r, *authCtl, user, false)
	userToken := tst.GetLoginToken(t, r, *authCtl, models.LoginRequestModel{Email: user.Email, Password: user.Password})
	tst.AssertStatusCode(t, do(http.MethodPost, "/api/v1/prompts/", userToken, valid), http.StatusUnauthorized)

	// plain admin may read but not write
	adminToken := CreateAdminAndGetToken(t, r, db, models.RoleAdmin)
	tst.AssertStatusCode(t, do(http.MethodPost, "/api/v1/prompts/", adminToken, valid), http.StatusForbidden)
	tst.AssertStatusCode(t, do(http.MethodGet, "/api/v1/prompts/", adminToken, nil), http.StatusOK)

	if v := maxVersion(); v != 0 {
		t.Fatalf("unauthorised requests created prompt versions: max version %d", v)
	}

	// superadmin: validation, then a valid create bumps the version by one
	superToken := CreateSuperAdminAndGetToken(t, r, db)
	tst.AssertStatusCode(t, do(http.MethodPost, "/api/v1/prompts/", superToken, map[string]string{"name": name, "template": ""}), http.StatusUnprocessableEntity)
	tst.AssertStatusCode(t, do(http.MethodPost, "/api/v1/prompts/", superToken, valid), http.StatusOK)
	if v := maxVersion(); v != 1 {
		t.Fatalf("expected version 1 after superadmin create, got %d", v)
	}
}
