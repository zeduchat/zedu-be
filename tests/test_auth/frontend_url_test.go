package test_auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestFrontendBaseURL(t *testing.T) {
	_ = tests.Setup()
	cfg := config.GetConfig()
	original := cfg.App.FRONTEND_URL
	t.Cleanup(func() { cfg.App.FRONTEND_URL = original })

	cases := map[string]string{
		"https://a.test":    "https://a.test/",
		"https://a.test/":   "https://a.test/",
		"https://a.test///": "https://a.test/",
	}
	for in, want := range cases {
		cfg.App.FRONTEND_URL = in
		if got := config.FrontendBaseURL(); got != want {
			t.Errorf("FrontendBaseURL() with %q = %q, want %q", in, got, want)
		}
	}
}

func TestMagicLinkIgnoresReferer(t *testing.T) {
	router, authController := SetupAuthTestRouter()
	db := authController.Db.Postgresql
	cfg := config.GetConfig()
	original := cfg.App.FRONTEND_URL
	cfg.App.FRONTEND_URL = "https://app.zedu.test"
	t.Cleanup(func() { cfg.App.FRONTEND_URL = original })

	hashed, _ := utility.HashPassword(utility.GenerateUUID())
	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "magic link referer user",
		Email:    "referer" + utility.GenerateUUID() + "@qa.team",
		Password: hashed,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	for _, referer := range []string{"https://evil.test/", ""} {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/magick-link",
			strings.NewReader(`{"email":"`+user.Email+`"}`))
		req.Header.Set("Content-Type", "application/json")
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		resp := httptestRecorder(router, req)
		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	}

	link := service.MagicLinkURL(config.FrontendBaseURL(), "t")
	if link != "https://app.zedu.test/auth/login/magic-link?token=t" {
		t.Fatalf("unexpected magic link %q", link)
	}
	if strings.Contains(link, "evil.test") {
		t.Fatalf("magic link contains the Referer host: %q", link)
	}
}

func httptestRecorder(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
