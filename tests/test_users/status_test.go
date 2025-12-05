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
