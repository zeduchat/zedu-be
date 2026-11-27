package test_auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func seedOTPUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	hashed, _ := utility.HashPassword(utility.GenerateUUID())
	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "otp purpose user",
		Email:    fmt.Sprintf("otp%v@qa.team", utility.GenerateUUID()),
		Password: hashed,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func seedOTP(t *testing.T, db *gorm.DB, email, purpose string) string {
	t.Helper()
	token := utility.GenerateUUID()
	row := models.PasswordReset{
		ID:        utility.GenerateUUID(),
		Email:     email,
		Purpose:   purpose,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed otp: %v", err)
	}
	return token
}

func postJSON(router *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestOTPPurposeIsolation(t *testing.T) {
	router, authController := SetupAuthTestRouter()
	db := authController.Db.Postgresql

	passwordOf := func(id string) string {
		var u models.User
		db.Select("password").Where("id = ?", id).First(&u)
		return u.Password
	}
	verifiedOf := func(id string) bool {
		var u models.User
		db.Select("is_verified").Where("id = ?", id).First(&u)
		return u.IsVerified
	}

	t.Run("verification code cannot reset the password", func(t *testing.T) {
		user := seedOTPUser(t, db)
		before := passwordOf(user.ID)
		token := seedOTP(t, db, user.Email, models.PurposeEmailVerification)

		resp := postJSON(router, "/api/v1/auth/password-reset/verify", models.ResetPasswordRequestModel{
			Email: user.Email, Token: token, NewPassword: "attackerpass",
		})
		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		if passwordOf(user.ID) != before {
			t.Fatal("password changed using an email-verification code")
		}
	})

	t.Run("reset code cannot verify the email", func(t *testing.T) {
		user := seedOTPUser(t, db)
		token := seedOTP(t, db, user.Email, models.PurposePasswordReset)

		resp := postJSON(router, "/api/v1/auth/email-request/verify", models.VerifyEmailTokenReqModel{
			Email: user.Email, Token: token,
		})
		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		if verifiedOf(user.ID) {
			t.Fatal("email verified using a password-reset code")
		}
	})

	t.Run("reset code for user A sent with user B's email is rejected", func(t *testing.T) {
		userA := seedOTPUser(t, db)
		userB := seedOTPUser(t, db)
		beforeA, beforeB := passwordOf(userA.ID), passwordOf(userB.ID)
		token := seedOTP(t, db, userA.Email, models.PurposePasswordReset)

		resp := postJSON(router, "/api/v1/auth/password-reset/verify", models.ResetPasswordRequestModel{
			Email: userB.Email, Token: token, NewPassword: "attackerpass",
		})
		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		if passwordOf(userA.ID) != beforeA || passwordOf(userB.ID) != beforeB {
			t.Fatal("password changed with a mismatched email")
		}
	})

	t.Run("verification code verifies the email", func(t *testing.T) {
		user := seedOTPUser(t, db)
		token := seedOTP(t, db, user.Email, models.PurposeEmailVerification)

		resp := postJSON(router, "/api/v1/auth/email-request/verify", models.VerifyEmailTokenReqModel{
			Email: user.Email, Token: token,
		})
		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		if !verifiedOf(user.ID) {
			t.Fatal("email not verified with a valid verification code")
		}
	})

	t.Run("missing email is a validation error", func(t *testing.T) {
		resp := postJSON(router, "/api/v1/auth/password-reset/verify", map[string]string{
			"token": "123456", "new_password": "newpassword",
		})
		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)

		resp = postJSON(router, "/api/v1/auth/email-request/verify", map[string]string{"token": "123456"})
		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	})
}
