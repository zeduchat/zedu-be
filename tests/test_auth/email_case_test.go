package test_auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestAuthEmailIsCaseInsensitive(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	email := fmt.Sprintf("MixedCase%v@qa.team", currUUID)
	normalisedEmail := strings.ToLower(email)

	userSignUpData := models.CreateUserRequestModel{
		Email:       email,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("case_user%v", currUUID),
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{
		Logger: logger,
		Test:   true,
	}}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, userSignUpData, false)
	r.POST("/api/v1/auth/login", authCtrl.LoginUser)
	r.POST("/api/v1/auth/password-reset", authCtrl.ResetPassword)
	r.POST("/api/v1/auth/magick-link", authCtrl.RequestMagicLink)

	defer func() {
		_ = db.Postgresql.Where("email = ?", normalisedEmail).Delete(&models.PasswordReset{}).Error
		_ = db.Postgresql.Where("email = ?", normalisedEmail).Delete(&models.MagicLink{}).Error
		_ = db.Postgresql.Where("email = ?", normalisedEmail).Delete(&models.User{}).Error
	}()

	mixedCaseEmail := strings.ToUpper(email[:1]) + email[1:]

	tests := []struct {
		Name         string
		Path         string
		RequestBody  any
		ExpectedCode int
	}{
		{
			Name:         "login with a capitalised email",
			Path:         "/api/v1/auth/login",
			RequestBody:  models.LoginRequestModel{Email: mixedCaseEmail, Password: userSignUpData.Password},
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "password reset with a capitalised email",
			Path:         "/api/v1/auth/password-reset",
			RequestBody:  models.ForgotPasswordRequestModel{Email: mixedCaseEmail},
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "magic link with a capitalised email",
			Path:         "/api/v1/auth/magick-link",
			RequestBody:  models.MagicLinkRequest{Email: mixedCaseEmail},
			ExpectedCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			if err := json.NewEncoder(&b).Encode(test.RequestBody); err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest(http.MethodPost, url.URL{Path: test.Path}.String(), &b)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Referer", "https://zedu.chat")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)
		})
	}
}
