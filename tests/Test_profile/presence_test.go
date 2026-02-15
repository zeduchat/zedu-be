package test_profile

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestPresenceFlow(t *testing.T) {
	_, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, err := utility.HashPassword("password")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Presence User",
		Email:    fmt.Sprintf("presence%v@qa.team", currUUID),
		Password: password,
	}

	db.Create(&regularUser)

	// Create profile for user if not created by hook (assuming hooks create it, otherwise manual)
	// Checking if profile exists
	var profile models.Profile
	if err := db.Where("userid = ?", regularUser.ID).First(&profile).Error; err != nil {
		// Create profile
		profile = models.Profile{
			ID:        utility.GenerateUUID(),
			Userid:    regularUser.ID,
			FirstName: "Presence",
			LastName:  "User",
			UserName:  "presence_user",
		}
		db.Create(&profile)
	}

	setup := func() (*gin.Engine, *auth.Controller) {
		router, profileController := SetupProfileTestRouter()
		authController := auth.Controller{
			Db:        profileController.Db,
			Validator: profileController.Validator,
			Logger:    profileController.Logger,
			ExtReq:    profileController.ExtReq,
		}
		return router, &authController
	}

	t.Run("Successfully Get Default Presence", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/profile/presence", nil)
		if err != nil {
			t.Fatalf("Failed to create new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		// Default should be online or empty depending on migration/model default
	})

	t.Run("Successfully Set Presence to Away", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		body := strings.NewReader(`{"presence": "away"}`)
		req, err := http.NewRequest(http.MethodPost, "/api/v1/profile/presence", body)
		if err != nil {
			t.Fatalf("Failed to create new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	})

	t.Run("Successfully Get Updated Presence", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/profile/presence", nil)
		if err != nil {
			t.Fatalf("Failed to create new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		// Response body should match "away"
		if !strings.Contains(resp.Body.String(), "away") {
			t.Errorf("Expected presence 'away', got %v", resp.Body.String())
		}
	})

	t.Run("Fail Set Invalid Presence", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		body := strings.NewReader(`{"presence": "invalid"}`)
		req, err := http.NewRequest(http.MethodPost, "/api/v1/profile/presence", body)
		if err != nil {
			t.Fatalf("Failed to create new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity) // Validation error
	})
}
