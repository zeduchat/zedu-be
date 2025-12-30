package test_users

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetMediaPreferences(t *testing.T) {
	router, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}

	userID, _ := uuid.FromString(regularUser.ID)

	// Create user-level media preferences
	mediaPreferences := models.MediaPreferences{
		ID:                    utility.GenerateUUID(),
		UserID:                userID,
		DeviceID:              nil,
		AutoDownloadPhotos:    "wifi_only",
		AutoDownloadAudio:     "always",
		AutoDownloadDocuments: "never",
		AutoDownloadVideos:    "wifi_only",
		UploadQuality:         "high",
	}

	// Create a device for testing device-specific preferences
	deviceID := utility.GenerateUUID()
	deviceUUID, _ := uuid.FromString(deviceID)
	device := models.DeviceRegistry{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: "Test Device",
		DeviceType: "mobile",
		IsActive:   true,
	}

	// Create device-specific preferences
	deviceMediaPreferences := models.MediaPreferences{
		ID:                    utility.GenerateUUID(),
		UserID:                userID,
		DeviceID:              &deviceUUID,
		AutoDownloadPhotos:    "always",
		AutoDownloadAudio:     "wifi_only",
		AutoDownloadDocuments: "always",
		AutoDownloadVideos:    "never",
		UploadQuality:         "original",
	}

	db.Create(&regularUser)
	db.Create(&device)
	db.Create(&mediaPreferences)
	db.Create(&deviceMediaPreferences)

	setup := func() (*gin.Engine, *auth.Controller) {
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("Successful Get User-Level Media Preferences", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/media-preferences", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences retrieved successfully")

		// Verify response data
		data := response["data"].(map[string]any)
		if data["auto_download_photos"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_photos to be 'wifi_only', got %v", data["auto_download_photos"])
		}
		if data["upload_quality"].(string) != "high" {
			t.Errorf("Expected upload_quality to be 'high', got %v", data["upload_quality"])
		}
	})

	t.Run("Successful Get Device-Specific Media Preferences", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/media-preferences?device_id=%s", deviceID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences retrieved successfully")

		// Verify device-specific preferences are returned
		data := response["data"].(map[string]any)
		if data["device_id"].(string) != deviceID {
			t.Errorf("Expected device_id to be %s, got %v", deviceID, data["device_id"])
		}
		if data["auto_download_photos"].(string) != "always" {
			t.Errorf("Expected auto_download_photos to be 'always', got %v", data["auto_download_photos"])
		}
		if data["upload_quality"].(string) != "original" {
			t.Errorf("Expected upload_quality to be 'original', got %v", data["upload_quality"])
		}
	})

	t.Run("Get Media Preferences Creates Defaults When None Exist", func(t *testing.T) {
		router, authController := setup()

		// Create a new user without preferences
		newUser := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "New User",
			Email:    fmt.Sprintf("newuser%v@qa.team", utility.GenerateUUID()),
			Password: password,
			Role:     int(models.RoleIdentity.User),
		}
		db.Create(&newUser)

		loginData := models.LoginRequestModel{
			Email:    newUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/media-preferences", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences retrieved successfully")

		// Verify default values are returned
		data := response["data"].(map[string]any)
		if data["auto_download_photos"].(string) != "wifi_only" {
			t.Errorf("Expected default auto_download_photos to be 'wifi_only', got %v", data["auto_download_photos"])
		}
		if data["upload_quality"].(string) != "standard" {
			t.Errorf("Expected default upload_quality to be 'standard', got %v", data["upload_quality"])
		}
	})

	t.Run("Invalid Device ID Format", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/media-preferences?device_id=invalid-uuid", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	})

	t.Run("Device Not Found", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		nonExistentDeviceID := utility.GenerateUUID()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/media-preferences?device_id=%s", nonExistentDeviceID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		response := tests.ParseResponse(resp)
		message := response["message"].(string)
		if message != "device not found: device not found" {
			t.Errorf("Expected error message about device not found, got %v", message)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		router, _ := setup()

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/media-preferences", nil)
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token could not be found!")
	})
}
