package test_users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestPatchUserStatus(t *testing.T) {
	_, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Status User",
		Email:    fmt.Sprintf("status_user_%s@qa.team", currUUID),
		Password: password,
	}

	db.Create(&user)
	db.Create(&models.Profile{
		ID:            utility.GenerateUUID(),
		Userid:        user.ID,
		Text:          "old",
		Icon:          ":old:",
		StatusTimeout: "0",
	})

	setup := func() (*gin.Engine, *auth.Controller) {
		router, userController := SetupUsersTestRouter()
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("successfully patches status", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		expiry := time.Now().Add(30 * time.Minute).Unix()
		payload := map[string]any{
			"text":   "In a meeting",
			"emoji":  ":spiral_calendar_pad:",
			"expiry": expiry,
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusOK)
		data := parsed["data"].(map[string]any)

		if got := data["text"].(string); got != payload["text"] {
			t.Fatalf("unexpected text: want %q got %q", payload["text"], got)
		}
		if got := data["emoji"].(string); got != payload["emoji"] {
			t.Fatalf("unexpected emoji: want %q got %q", payload["emoji"], got)
		}
		if got := int64(data["expiry"].(float64)); got != expiry {
			t.Fatalf("unexpected expiry: want %d got %d", expiry, got)
		}

		var updated models.Profile
		if err := db.Where("userid = ?", user.ID).First(&updated).Error; err != nil {
			t.Fatalf("failed to fetch updated profile: %v", err)
		}
		if updated.Text != payload["text"] || updated.Icon != payload["emoji"] {
			t.Fatalf("profile not updated; got text=%q icon=%q", updated.Text, updated.Icon)
		}
		if updated.StatusTimeout != fmt.Sprintf("%d", expiry) {
			t.Fatalf("status_timeout not updated; got %q want %d", updated.StatusTimeout, expiry)
		}
	})

	t.Run("invalid payload returns bad request", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		payload := map[string]any{
			"emoji":  "bad emoji", // whitespace not allowed
			"expiry": -5,
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusBadRequest)
	})
}

func TestGetUserStatus(t *testing.T) {
	_, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Status Get User",
		Email:    fmt.Sprintf("status_get_user_%s@qa.team", currUUID),
		Password: password,
	}

	otherUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Other User",
		Email:    fmt.Sprintf("other_user_%s@qa.team", currUUID),
		Password: password,
	}

	db.Create(&user)
	db.Create(&otherUser)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, userController := SetupUsersTestRouter()
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("successfully retrieves status with all fields", func(t *testing.T) {
		router, authController := setup()

		expiry := time.Now().Add(30 * time.Minute).Unix()
		db.Create(&models.Profile{
			ID:               utility.GenerateUUID(),
			Userid:           user.ID,
			Text:             "In a meeting",
			Icon:             ":spiral_calendar_pad:",
			StatusTimeout:    fmt.Sprintf("%d", expiry),
			StatusVisibility: "public",
		})

		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/status", user.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusOK)
		data := parsed["data"].(map[string]any)

		if got := data["text"].(string); got != "In a meeting" {
			t.Fatalf("unexpected text: want %q got %q", "In a meeting", got)
		}
		if got := data["emoji"].(string); got != ":spiral_calendar_pad:" {
			t.Fatalf("unexpected emoji: want %q got %q", ":spiral_calendar_pad:", got)
		}
		if got := int64(data["expiry"].(float64)); got != expiry {
			t.Fatalf("unexpected expiry: want %d got %d", expiry, got)
		}
		if got := data["visibility"].(string); got != "public" {
			t.Fatalf("unexpected visibility: want %q got %q", "public", got)
		}
	})

	t.Run("returns empty defaults when no status is set", func(t *testing.T) {
		router, authController := setup()

		userNoStatus := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "No Status User",
			Email:    fmt.Sprintf("no_status_user_%s@qa.team", utility.GenerateUUID()),
			Password: password,
		}
		db.Create(&userNoStatus)
		db.Create(&models.Profile{
			ID:     utility.GenerateUUID(),
			Userid: userNoStatus.ID,
		})

		loginData := models.LoginRequestModel{
			Email:    userNoStatus.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/status", userNoStatus.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusOK)
		data := parsed["data"].(map[string]any)

		if got := data["text"].(string); got != "" {
			t.Fatalf("unexpected text: want empty string got %q", got)
		}
		if got := data["emoji"].(string); got != "" {
			t.Fatalf("unexpected emoji: want empty string got %q", got)
		}
		if got := int64(data["expiry"].(float64)); got != 0 {
			t.Fatalf("unexpected expiry: want 0 got %d", got)
		}
		if got := data["visibility"].(string); got != "public" {
			t.Fatalf("unexpected visibility: want %q got %q", "public", got)
		}
	})

	t.Run("returns empty defaults when profile not found", func(t *testing.T) {
		router, authController := setup()

		userNoProfile := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "No Profile User",
			Email:    fmt.Sprintf("no_profile_user_%s@qa.team", utility.GenerateUUID()),
			Password: password,
		}
		db.Create(&userNoProfile)
		// Don't create profile

		loginData := models.LoginRequestModel{
			Email:    userNoProfile.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/status", userNoProfile.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusOK)
		data := parsed["data"].(map[string]any)

		if got := data["text"].(string); got != "" {
			t.Fatalf("unexpected text: want empty string got %q", got)
		}
		if got := data["visibility"].(string); got != "public" {
			t.Fatalf("unexpected visibility: want %q got %q", "public", got)
		}
	})

	t.Run("returns forbidden when accessing another user's status", func(t *testing.T) {
		router, authController := setup()

		db.Create(&models.Profile{
			ID:     utility.GenerateUUID(),
			Userid: otherUser.ID,
		})

		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/status", otherUser.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusForbidden)
	})

	t.Run("returns bad request for invalid UUID format", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid/status", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusBadRequest)
	})

	t.Run("returns unauthorized when no token provided", func(t *testing.T) {
		router, _ := setup()

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/status", user.ID), nil)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
	})

	t.Run("retrieves status with custom visibility", func(t *testing.T) {
		router, authController := setup()

		userCustomVisibility := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "Custom Visibility User",
			Email:    fmt.Sprintf("custom_visibility_user_%s@qa.team", utility.GenerateUUID()),
			Password: password,
		}
		db.Create(&userCustomVisibility)

		expiry := time.Now().Add(1 * time.Hour).Unix()
		db.Create(&models.Profile{
			ID:               utility.GenerateUUID(),
			Userid:           userCustomVisibility.ID,
			Text:             "Working remotely",
			Icon:             ":house:",
			StatusTimeout:    fmt.Sprintf("%d", expiry),
			StatusVisibility: "workspace",
		})

		loginData := models.LoginRequestModel{
			Email:    userCustomVisibility.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/status", userCustomVisibility.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["visibility"].(string); got != "workspace" {
			t.Fatalf("unexpected visibility: want %q got %q", "workspace", got)
		}
	})
}
