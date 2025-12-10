package test_users

import (
	"bytes"
	"encoding/json"
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

func TestResetAutoDownloadSettings(t *testing.T) {
	_, userController := SetupUsersTestRouter()
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

	// Create user-level media preferences with custom auto-download settings
	mediaPreferences := models.MediaPreferences{
		ID:                    utility.GenerateUUID(),
		UserID:                userID,
		DeviceID:              nil,
		AutoDownloadPhotos:    "always",
		AutoDownloadAudio:     "never",
		AutoDownloadDocuments: "always",
		AutoDownloadVideos:    "never",
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

	// Create device-specific preferences with custom auto-download settings
	deviceMediaPreferences := models.MediaPreferences{
		ID:                    utility.GenerateUUID(),
		UserID:                userID,
		DeviceID:              &deviceUUID,
		AutoDownloadPhotos:    "always",
		AutoDownloadAudio:     "never",
		AutoDownloadDocuments: "always",
		AutoDownloadVideos:    "never",
		UploadQuality:         "original",
	}

	db.Create(&regularUser)
	db.Create(&device)
	db.Create(&mediaPreferences)
	db.Create(&deviceMediaPreferences)

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

	t.Run("Successful Reset User-Level Auto-Download Settings", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		// Reset without device_id (user-level)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Auto-download settings reset successfully")

		// Verify all auto-download settings are reset to "wifi_only"
		data := response["data"].(map[string]any)
		if data["auto_download_photos"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_photos to be 'wifi_only', got %v", data["auto_download_photos"])
		}
		if data["auto_download_audio"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_audio to be 'wifi_only', got %v", data["auto_download_audio"])
		}
		if data["auto_download_documents"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_documents to be 'wifi_only', got %v", data["auto_download_documents"])
		}
		if data["auto_download_videos"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_videos to be 'wifi_only', got %v", data["auto_download_videos"])
		}
		// Verify upload_quality is not affected
		if data["upload_quality"].(string) != "high" {
			t.Errorf("Expected upload_quality to remain 'high', got %v", data["upload_quality"])
		}
		// Verify device_id is null for user-level preferences
		if data["device_id"] != nil {
			t.Errorf("Expected device_id to be null for user-level preferences, got %v", data["device_id"])
		}
	})

	t.Run("Successful Reset Device-Specific Auto-Download Settings", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		// Reset with device_id
		resetData := models.ResetAutoDownloadRequest{
			DeviceID: &deviceID,
		}
		body, _ := json.Marshal(resetData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Auto-download settings reset successfully")

		// Verify all auto-download settings are reset to "wifi_only"
		data := response["data"].(map[string]any)
		if data["auto_download_photos"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_photos to be 'wifi_only', got %v", data["auto_download_photos"])
		}
		if data["auto_download_audio"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_audio to be 'wifi_only', got %v", data["auto_download_audio"])
		}
		if data["auto_download_documents"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_documents to be 'wifi_only', got %v", data["auto_download_documents"])
		}
		if data["auto_download_videos"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_videos to be 'wifi_only', got %v", data["auto_download_videos"])
		}
		// Verify upload_quality is not affected
		if data["upload_quality"].(string) != "original" {
			t.Errorf("Expected upload_quality to remain 'original', got %v", data["upload_quality"])
		}
		// Verify device_id matches
		if data["device_id"].(string) != deviceID {
			t.Errorf("Expected device_id to be %s, got %v", deviceID, data["device_id"])
		}
	})

	t.Run("Reset Creates Device Preferences If They Don't Exist", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		// Create a new device without preferences
		newDeviceID := utility.GenerateUUID()
		newDevice := models.DeviceRegistry{
			ID:         newDeviceID,
			UserID:     userID,
			DeviceName: "New Device",
			DeviceType: "desktop",
			IsActive:   true,
		}
		db.Create(&newDevice)

		// Reset for this device (should create preferences)
		resetData := models.ResetAutoDownloadRequest{
			DeviceID: &newDeviceID,
		}
		body, _ := json.Marshal(resetData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Auto-download settings reset successfully")

		// Verify preferences were created with default values
		data := response["data"].(map[string]any)
		if data["device_id"].(string) != newDeviceID {
			t.Errorf("Expected device_id to be %s, got %v", newDeviceID, data["device_id"])
		}
		if data["auto_download_photos"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_photos to be 'wifi_only', got %v", data["auto_download_photos"])
		}
	})

	t.Run("Reset Creates User Preferences If They Don't Exist", func(t *testing.T) {
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
		token := tests.GetLoginToken(t, router, *authController, loginData)

		// Reset (should create preferences)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Auto-download settings reset successfully")

		// Verify preferences were created with default values
		data := response["data"].(map[string]any)
		if data["auto_download_photos"].(string) != "wifi_only" {
			t.Errorf("Expected auto_download_photos to be 'wifi_only', got %v", data["auto_download_photos"])
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
		token := tests.GetLoginToken(t, router, *authController, loginData)

		invalidDeviceID := "invalid-uuid"
		resetData := models.ResetAutoDownloadRequest{
			DeviceID: &invalidDeviceID,
		}
		body, _ := json.Marshal(resetData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	})

	t.Run("Device Not Found", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		nonExistentDeviceID := utility.GenerateUUID()
		resetData := models.ResetAutoDownloadRequest{
			DeviceID: &nonExistentDeviceID,
		}
		body, _ := json.Marshal(resetData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", bytes.NewReader(body))
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

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", nil)
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token could not be found!")
	})

	t.Run("Reset With Empty Body (User-Level)", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		// Send empty JSON body
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/media-preferences/reset-autodownload", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Auto-download settings reset successfully")

		// Verify user-level preferences were reset
		data := response["data"].(map[string]any)
		if data["device_id"] != nil {
			t.Errorf("Expected device_id to be null for user-level reset, got %v", data["device_id"])
		}
	})
}

