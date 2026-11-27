package test_auth

import (
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestAllowRate(t *testing.T) {
	_, authController := SetupAuthTestRouter()
	rdb := authController.Db.Redis
	if rdb == nil {
		t.Fatal("test redis is not connected")
	}

	key := "test:" + utility.GenerateUUID()
	for i := 1; i <= 3; i++ {
		if !middleware.AllowRate(rdb, key, 3, time.Minute) {
			t.Fatalf("call %d should be allowed", i)
		}
	}
	if middleware.AllowRate(rdb, key, 3, time.Minute) {
		t.Fatal("call 4 should be denied")
	}
	if middleware.AllowRate(nil, key, 3, time.Minute) {
		t.Fatal("nil redis client must fail closed")
	}
}

func TestOTPVerifyRateLimitByIP(t *testing.T) {
	router, authController := SetupAuthTestRouter()
	router.POST("/limited/password-reset/verify",
		middleware.RateLimitByIP(authController.Db.Redis, "t-"+utility.GenerateUUID(), 3, time.Minute),
		authController.VerifyResetToken)

	body := models.ResetPasswordRequestModel{Email: "nobody@qa.team", Token: "000000", NewPassword: "newpassword"}
	for i := 1; i <= 3; i++ {
		resp := postJSON(router, "/limited/password-reset/verify", body)
		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
	}
	resp := postJSON(router, "/limited/password-reset/verify", body)
	tests.AssertStatusCode(t, resp.Code, http.StatusTooManyRequests)
}

func TestOTPAttemptCap(t *testing.T) {
	router, authController := SetupAuthTestRouter()
	db := authController.Db.Postgresql

	user := seedOTPUser(t, db)
	token := seedOTP(t, db, user.Email, models.PurposePasswordReset)

	for i := 0; i < models.MaxOTPAttempts; i++ {
		resp := postJSON(router, "/api/v1/auth/password-reset/verify", models.ResetPasswordRequestModel{
			Email: user.Email, Token: "wrong-" + utility.GenerateUUID(), NewPassword: "newpassword",
		})
		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
	}

	resp := postJSON(router, "/api/v1/auth/password-reset/verify", models.ResetPasswordRequestModel{
		Email: user.Email, Token: token, NewPassword: "newpassword",
	})
	tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)

	var count int64
	db.Model(&models.PasswordReset{}).Where("email = ?", user.Email).Count(&count)
	if count != 0 {
		t.Fatalf("code should be deleted after %d failures, %d row(s) left", models.MaxOTPAttempts, count)
	}
}

func TestOTPRequestCooldown(t *testing.T) {
	router, authController := SetupAuthTestRouter()
	user := seedOTPUser(t, authController.Db.Postgresql)

	body := models.ForgotPasswordRequestModel{Email: user.Email}
	resp := postJSON(router, "/api/v1/auth/password-reset", body)
	tests.AssertStatusCode(t, resp.Code, http.StatusOK)

	resp = postJSON(router, "/api/v1/auth/password-reset", body)
	tests.AssertStatusCode(t, resp.Code, http.StatusTooManyRequests)
}

func TestMagicLinkTokenIsLong(t *testing.T) {
	router, authController := SetupAuthTestRouter()
	db := authController.Db.Postgresql
	user := seedOTPUser(t, db)

	resp := postJSON(router, "/api/v1/auth/magick-link", models.MagicLinkRequest{Email: user.Email})
	tests.AssertStatusCode(t, resp.Code, http.StatusOK)

	var magic models.MagicLink
	if err := db.Where("email = ?", user.Email).First(&magic).Error; err != nil {
		t.Fatalf("magic link row: %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(magic.Token) {
		t.Fatalf("magic link token should be 64 hex characters, got length %d", len(magic.Token))
	}

	resp = postJSON(router, "/api/v1/auth/magick-link/verify", models.VerifyMagicLinkRequest{Token: magic.Token})
	tests.AssertStatusCode(t, resp.Code, http.StatusOK)
}
