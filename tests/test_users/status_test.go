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
	router, userController := SetupUsersTestRouter()
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
		Icon:          "😊",
		StatusTimeout: "0",
	})

	setup := func() (*gin.Engine, *auth.Controller) {
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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		expiry := time.Now().Add(30 * time.Minute).Unix()
		payload := map[string]any{
			"text":   "In a meeting",
			"emoji":  "📅",
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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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
	router, userController := SetupUsersTestRouter()
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
			Icon:             "📅",
			StatusTimeout:    fmt.Sprintf("%d", expiry),
			StatusVisibility: "public",
		})

		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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
		if got := data["emoji"].(string); got != "📅" {
			t.Fatalf("unexpected emoji: want %q got %q", "📅", got)
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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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
			Icon:             "🏠",
			StatusTimeout:    fmt.Sprintf("%d", expiry),
			StatusVisibility: "workspace",
		})

		loginData := models.LoginRequestModel{
			Email:    userCustomVisibility.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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

func TestSetUserStatus(t *testing.T) {
	router, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql
	t.Cleanup(func() {
		tests.Cleanup(userController.Db)
	})

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Set Status User",
		Email:    fmt.Sprintf("set_status_user_%s@qa.team", currUUID),
		Password: password,
	}

	otherUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Other Set Status User",
		Email:    fmt.Sprintf("other_set_status_user_%s@qa.team", currUUID),
		Password: password,
	}

	db.Create(&user)
	db.Create(&otherUser)
	db.Create(&models.Profile{
		ID:            utility.GenerateUUID(),
		Userid:        user.ID,
		Text:          "old status",
		Icon:          "😊",
		StatusTimeout: "0",
	})

	setup := func() (*gin.Engine, *auth.Controller) {
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("successfully sets status with all fields", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		expiry := time.Now().Add(30 * time.Minute).Unix()
		payload := map[string]any{
			"text":       "In a meeting",
			"emoji":      "📅",
			"expiry":     expiry,
			"visibility": "public",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusCreated)
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
		if got := data["visibility"].(string); got != payload["visibility"] {
			t.Fatalf("unexpected visibility: want %q got %q", payload["visibility"], got)
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
		if updated.StatusVisibility != payload["visibility"] {
			t.Fatalf("status_visibility not updated; got %q want %q", updated.StatusVisibility, payload["visibility"])
		}
	})

	t.Run("successfully sets status with only required text field", func(t *testing.T) {
		router, authController := setup()

		userMinimal := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "Minimal Status User",
			Email:    fmt.Sprintf("minimal_status_user_%s@qa.team", utility.GenerateUUID()),
			Password: password,
		}
		db.Create(&userMinimal)
		db.Create(&models.Profile{
			ID:     utility.GenerateUUID(),
			Userid: userMinimal.ID,
		})

		loginData := models.LoginRequestModel{
			Email:    userMinimal.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "Working on a project",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", userMinimal.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusCreated)
		data := parsed["data"].(map[string]any)

		if got := data["text"].(string); got != payload["text"] {
			t.Fatalf("unexpected text: want %q got %q", payload["text"], got)
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

	t.Run("successfully sets status with different visibility options", func(t *testing.T) {
		router, authController := setup()
		userContacts := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "Contacts Visibility User",
			Email:    fmt.Sprintf("contacts_visibility_user_%s@qa.team", utility.GenerateUUID()),
			Password: password,
		}

		db := authController.Db.Postgresql
		db.Create(&userContacts)
		db.Create(&models.Profile{
			ID:     utility.GenerateUUID(),
			Userid: userContacts.ID,
		})

		loginData := models.LoginRequestModel{
			Email:    userContacts.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":       "Available for contacts",
			"visibility": "contacts",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", userContacts.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["visibility"].(string); got != "contacts" {
			t.Fatalf("unexpected visibility: want %q got %q", "contacts", got)
		}

		var updated models.Profile
		if err := db.Where("userid = ?", userContacts.ID).First(&updated).Error; err != nil {
			t.Fatalf("failed to fetch updated profile: %v", err)
		}
		if updated.StatusVisibility != "contacts" {
			t.Fatalf("status_visibility not updated; got %q want %q", updated.StatusVisibility, "contacts")
		}
	})

	t.Run("returns bad request when text is empty", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns bad request when text is missing", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"emoji": "😊",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns bad request when text exceeds 255 characters", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		longText := make([]byte, 256)
		for i := range longText {
			longText[i] = 'a'
		}
		payload := map[string]any{
			"text": string(longText),
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns bad request when emoji contains whitespace", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":  "Status with bad emoji",
			"emoji": "bad emoji",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns bad request when emoji exceeds 64 characters", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		longEmoji := make([]byte, 65)
		for i := range longEmoji {
			longEmoji[i] = 'e'
		}
		payload := map[string]any{
			"text":  "Status with long emoji",
			"emoji": string(longEmoji),
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns bad request when expiry is negative", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":   "Status with negative expiry",
			"expiry": -1,
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns bad request when visibility is invalid", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":       "Status with invalid visibility",
			"visibility": "invalid",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns forbidden when setting another user's status", func(t *testing.T) {
		router, authController := setup()

		db.Create(&models.Profile{
			ID:     utility.GenerateUUID(),
			Userid: otherUser.ID,
		})

		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "Trying to set other user's status",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", otherUser.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "Status text",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/invalid-uuid/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusBadRequest)
	})

	t.Run("returns unauthorized when no token provided", func(t *testing.T) {
		router, _ := setup()

		payload := map[string]any{
			"text": "Status text",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
	})

	t.Run("returns not found when profile does not exist", func(t *testing.T) {
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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "Status text",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", userNoProfile.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusNotFound)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusNotFound)
	})

	t.Run("trims whitespace from text", func(t *testing.T) {
		router, authController := setup()
		userTrimmed := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "Trimmed Text User",
			Email:    fmt.Sprintf("trimmed_text_user_%s@qa.team", utility.GenerateUUID()),
			Password: password,
		}
		db.Create(&userTrimmed)
		db.Create(&models.Profile{
			ID:     utility.GenerateUUID(),
			Userid: userTrimmed.ID,
		})

		loginData := models.LoginRequestModel{
			Email:    userTrimmed.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "  Trimmed text  ",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", userTrimmed.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["text"].(string); got != "Trimmed text" {
			t.Fatalf("unexpected text: want %q got %q", "Trimmed text", got)
		}
	})
}

func TestEmojiValidation(t *testing.T) {
	_, userController := SetupUsersTestRouter()
	t.Cleanup(func() {
		tests.Cleanup(userController.Db)
	})
	db := userController.Db.Postgresql

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Emoji Test User",
		Email:    fmt.Sprintf("emoji_test_user_%s@qa.team", currUUID),
		Password: password,
	}

	db.Create(&user)
	db.Create(&models.Profile{
		ID:     utility.GenerateUUID(),
		Userid: user.ID,
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

	t.Run("successfully sets status with valid Unicode emoji", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":  "Status with emoji",
			"emoji": "😊",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["emoji"].(string); got != "😊" {
			t.Fatalf("unexpected emoji: want %q got %q", "😊", got)
		}
	})

	t.Run("successfully sets status with emoji sequence", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":  "Status with emoji sequence",
			"emoji": "👨‍👩‍👧‍👦",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["emoji"].(string); got != "👨‍👩‍👧‍👦" {
			t.Fatalf("unexpected emoji: want %q got %q", "👨‍👩‍👧‍👦", got)
		}
	})

	t.Run("successfully sets status with emoji and skin tone", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":  "Status with skin tone emoji",
			"emoji": "👍🏿",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["emoji"].(string); got != "👍🏿" {
			t.Fatalf("unexpected emoji: want %q got %q", "👍🏿", got)
		}
	})

	t.Run("returns bad request when emoji is invalid (non-emoji string)", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":  "Status with invalid emoji",
			"emoji": "notanemoji",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// SetUserStatus uses go-playground validator which returns 422 (Unprocessable Entity) for validation errors
		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("returns bad request when emoji contains mixed emoji and text", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":  "Status with mixed emoji",
			"emoji": "😊hello",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// SetUserStatus uses go-playground validator which returns 422 (Unprocessable Entity) for validation errors
		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		parsed := tests.ParseResponse(resp)
		tests.AssertStatusCode(t, int(parsed["status_code"].(float64)), http.StatusUnprocessableEntity)
	})

	t.Run("successfully patches status with valid emoji", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"emoji": "🎉",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["emoji"].(string); got != "🎉" {
			t.Fatalf("unexpected emoji: want %q got %q", "🎉", got)
		}
	})

	t.Run("returns bad request when patching with invalid emoji", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"emoji": "invalid123",
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

	t.Run("allows empty emoji (optional field)", func(t *testing.T) {
		router, authController := setup()
		t.Cleanup(func() {
			tests.Cleanup(authController.Db)
		})
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "Status without emoji",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		parsed := tests.ParseResponse(resp)
		data := parsed["data"].(map[string]any)

		if got := data["emoji"].(string); got != "" {
			t.Fatalf("unexpected emoji: want empty string got %q", got)
		}
	})
}
