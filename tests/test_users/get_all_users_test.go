package test_users

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/user"
	"github.com/hngprojects/telex_be/pkg/middleware"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/tests/test_admin"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetAllUsersIsAdminOnly(t *testing.T) {
	logger := tst.Setup()
	// The admin test router has the backoffice login route.
	r, _, _, db := test_admin.SetupAdminTestRouter()

	userCtrl := user.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r.GET("/api/v1/users", middleware.AdminAuthorize(db.Postgresql), userCtrl.GetAllUsers)

	currUUID := utility.GenerateUUID()
	authCtrl := auth.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, signup, false)
	userToken := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})
	adminToken := test_admin.CreateAdminAndGetToken(t, r, db, models.RoleAdmin)

	cases := []struct {
		name  string
		token string
		query string
		want  int
	}{
		{"normal user token", userToken, "", http.StatusUnauthorized},
		{"no token", "", "", http.StatusUnauthorized},
		{"admin token", adminToken, "", http.StatusOK},
		{"admin with oversized limit", adminToken, "?limit=10000", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/users"+tc.query, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("got %d want %d: %s", rr.Code, tc.want, rr.Body.String())
			}
			body := tst.ParseResponse(rr)
			users, _ := body["data"].([]any)
			if tc.want != http.StatusOK && len(users) > 0 {
				t.Errorf("user list leaked in a %d response", rr.Code)
			}
			if len(users) > 100 {
				t.Errorf("limit not capped: got %d users", len(users))
			}
		})
	}
}
