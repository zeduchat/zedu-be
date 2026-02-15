package test_profile

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func setup() (*gin.Engine, *auth.Controller) {
	router, profileController := SetupProfileTestRouter()
	authController := auth.Controller{
		Db:        profileController.Db,
		Validator: profileController.Validator,
		Logger:    profileController.Logger,
		ExtReq:    profileController.ExtReq,
	}
	return router, &authController
}

func TestPresenceFlow(t *testing.T) {
	router, authController := setup()
	db := storage.Connection().Postgresql

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password123")

	// Create a user
	user := models.User{
		ID:       currUUID,
		Email:    fmt.Sprintf("presence_test_%s@example.com", currUUID),
		Password: password,
		Name:     "Test User",
	}
	db.Create(&user)
	defer db.Unscoped().Delete(&user)

	// Create a profile for this user
	profile := models.Profile{
		ID:        utility.GenerateUUID(),
		Userid:    user.ID,
		IsActive:  false,
		FirstName: "Test",
		LastName:  "User",
		UserName:  fmt.Sprintf("testuser_%s", currUUID),
	}
	db.Create(&profile)
	defer db.Unscoped().Delete(&profile)

	// Login to get token
	loginData := models.LoginRequestModel{
		Email:    user.Email,
		Password: "password123",
	}
	token := tests.GetLoginToken(t, router, *authController, loginData)

	t.Run("Successfully change presence to online", func(t *testing.T) {
		reqBody := `{"is_active": true}`
		req, _ := http.NewRequest("POST", "/api/v1/profile/presence", bytes.NewBuffer([]byte(reqBody)))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedProfile models.Profile
		db.Where("id = ?", profile.ID).First(&updatedProfile)
		assert.Equal(t, true, updatedProfile.IsActive)
	})

	t.Run("Successfully change presence to offline", func(t *testing.T) {
		reqBody := `{"is_active": false}`
		req, _ := http.NewRequest("POST", "/api/v1/profile/presence", bytes.NewBuffer([]byte(reqBody)))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedProfile models.Profile
		db.Where("id = ?", profile.ID).First(&updatedProfile)
		assert.Equal(t, false, updatedProfile.IsActive)
	})

	t.Run("Successfully get own presence", func(t *testing.T) {
		// Set to online first
		db.Model(&profile).Update("is_active", true)

		req, _ := http.NewRequest("GET", "/api/v1/profile/presence", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"is_active":true`)
	})

	t.Run("Successfully get other user presence", func(t *testing.T) {
		// Set to online first
		db.Model(&profile).Update("is_active", true)

		req, _ := http.NewRequest("GET", "/api/v1/profile/presence/"+user.ID, nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"is_active":true`)
	})

	t.Run("Fail with invalid body", func(t *testing.T) {
		reqBody := `{"is_active": "invalid"}` // Invalid type
		req, _ := http.NewRequest("POST", "/api/v1/profile/presence", bytes.NewBuffer([]byte(reqBody)))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
