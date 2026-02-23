package test_users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestSetUserStatusWithExpirySchedulesJob(t *testing.T) {
	router, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Status Expiry User",
		Email:    fmt.Sprintf("status_expiry_%s@qa.team", currUUID),
		Password: password,
	}

	db.Create(&user)
	db.Create(&models.Profile{
		ID:     utility.GenerateUUID(),
		Userid: user.ID,
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

	t.Run("schedules river job when expiry is set", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":   "In a meeting",
			"emoji":  "📅",
			"expiry": "30 minutes",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)

		var profile models.Profile
		if err := db.Where("userid = ?", user.ID).First(&profile).Error; err != nil {
			t.Fatalf("failed to fetch profile: %v", err)
		}

		if profile.RiverJobID == nil {
			t.Fatalf("expected river_job_id to be set, got nil")
		}
		if profile.StatusTimeout == "" {
			t.Fatalf("expected status_timeout to be set, got empty string")
		}
	})

	t.Run("clears river_job_id when expiry is zero", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text": "Working on something",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)

		var profile models.Profile
		if err := db.Where("userid = ?", user.ID).First(&profile).Error; err != nil {
			t.Fatalf("failed to fetch profile: %v", err)
		}

		if profile.RiverJobID != nil {
			t.Fatalf("expected river_job_id to be nil, got %d", *profile.RiverJobID)
		}
	})

	t.Run("replaces job when updating expiry", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		payload := map[string]any{
			"text":   "Initial status",
			"expiry": "30 minutes",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)

		var profile models.Profile
		if err := db.Where("userid = ?", user.ID).First(&profile).Error; err != nil {
			t.Fatalf("failed to fetch profile: %v", err)
		}

		firstJobID := profile.RiverJobID
		if firstJobID == nil {
			t.Fatalf("expected first job_id to be set")
		}

		payload = map[string]any{
			"text":   "Updated status",
			"expiry": "1 hour",
		}
		body, _ = json.Marshal(payload)

		req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%s/status", user.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)

		if err := db.Where("userid = ?", user.ID).First(&profile).Error; err != nil {
			t.Fatalf("failed to fetch profile: %v", err)
		}

		if profile.RiverJobID == nil {
			t.Fatalf("expected new job_id to be set")
		}
		if *profile.RiverJobID == *firstJobID {
			t.Fatalf("expected job_id to change, got same value %d", *firstJobID)
		}
	})
}
