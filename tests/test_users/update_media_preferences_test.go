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

func TestUpdateMediaPreferences(t *testing.T) {
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
		UploadQuality:         "standard",
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

	db.Create(&regularUser)
	db.Create(&device)
	db.Create(&mediaPreferences)

	setup := func() (*gin.Engine, *auth.Controller) {
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("Successful Update User-Level Media Preferences - Full Update", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := models.UpdateMediaPreferencesRequest{
			AutoDownloadPhotos:    "always",
			AutoDownloadAudio:     "never",
			AutoDownloadDocuments: "wifi_only",
			AutoDownloadVideos:    "always",
			UploadQuality:         "high",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences updated successfully")

		// Verify updated values
		data := response["data"].(map[string]any)
		if data["auto_download_photos"].(string) != "always" {
			t.Errorf("Expected auto_download_photos to be 'always', got %v", data["auto_download_photos"])
		}
		if data["upload_quality"].(string) != "high" {
			t.Errorf("Expected upload_quality to be 'high', got %v", data["upload_quality"])
		}
	})

	t.Run("Successful Update User-Level Media Preferences - Partial Update", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		// Only update upload quality
		updateData := models.UpdateMediaPreferencesRequest{
			UploadQuality: "original",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences updated successfully")

		// Verify only upload_quality was updated
		data := response["data"].(map[string]any)
		if data["upload_quality"].(string) != "original" {
			t.Errorf("Expected upload_quality to be 'original', got %v", data["upload_quality"])
		}
	})

	t.Run("Successful Create Device-Specific Media Preferences", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		// Create new device
		newDeviceID := utility.GenerateUUID()
		newDevice := models.DeviceRegistry{
			ID:         newDeviceID,
			UserID:     userID,
			DeviceName: "New Test Device",
			DeviceType: "desktop",
			IsActive:   true,
		}
		db.Create(&newDevice)

		updateData := models.UpdateMediaPreferencesRequest{
			DeviceID:              &newDeviceID,
			AutoDownloadPhotos:    "never",
			AutoDownloadAudio:     "always",
			AutoDownloadDocuments: "wifi_only",
			AutoDownloadVideos:    "never",
			UploadQuality:         "standard",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences updated successfully")

		// Verify device-specific preferences were created
		data := response["data"].(map[string]any)
		if data["device_id"].(string) != newDeviceID {
			t.Errorf("Expected device_id to be %s, got %v", newDeviceID, data["device_id"])
		}
		if data["auto_download_photos"].(string) != "never" {
			t.Errorf("Expected auto_download_photos to be 'never', got %v", data["auto_download_photos"])
		}
	})

	t.Run("Successful Update Device-Specific Media Preferences", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		// Create device-specific preferences first
		devicePref := models.MediaPreferences{
			ID:                    utility.GenerateUUID(),
			UserID:                userID,
			DeviceID:              &deviceUUID,
			AutoDownloadPhotos:    "wifi_only",
			AutoDownloadAudio:     "wifi_only",
			AutoDownloadDocuments: "wifi_only",
			AutoDownloadVideos:    "wifi_only",
			UploadQuality:         "standard",
		}
		db.Create(&devicePref)

		// Update device-specific preferences
		updateData := models.UpdateMediaPreferencesRequest{
			DeviceID:           &deviceID,
			AutoDownloadPhotos: "always",
			UploadQuality:      "high",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences updated successfully")

		// Verify updated values
		data := response["data"].(map[string]any)
		if data["device_id"].(string) != deviceID {
			t.Errorf("Expected device_id to be %s, got %v", deviceID, data["device_id"])
		}
		if data["auto_download_photos"].(string) != "always" {
			t.Errorf("Expected auto_download_photos to be 'always', got %v", data["auto_download_photos"])
		}
		if data["upload_quality"].(string) != "high" {
			t.Errorf("Expected upload_quality to be 'high', got %v", data["upload_quality"])
		}
	})

	t.Run("Invalid Upload Quality Value", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := map[string]string{
			"upload_quality": "invalid_quality",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	})

	t.Run("Invalid Auto-Download Value", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := map[string]string{
			"auto_download_photos": "invalid_option",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	})

	t.Run("Invalid Device ID Format", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		invalidDeviceID := "invalid-uuid"
		updateData := models.UpdateMediaPreferencesRequest{
			DeviceID: &invalidDeviceID,
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
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
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		nonExistentDeviceID := utility.GenerateUUID()
		updateData := models.UpdateMediaPreferencesRequest{
			DeviceID:      &nonExistentDeviceID,
			UploadQuality: "high",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
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

		updateData := models.UpdateMediaPreferencesRequest{
			UploadQuality: "high",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token could not be found!")
	})

	t.Run("Create User Preferences When None Exist", func(t *testing.T) {
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

		updateData := models.UpdateMediaPreferencesRequest{
			AutoDownloadPhotos: "always",
			UploadQuality:      "high",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Media preferences updated successfully")

		// Verify preferences were created with updated values
		data := response["data"].(map[string]any)
		if data["auto_download_photos"].(string) != "always" {
			t.Errorf("Expected auto_download_photos to be 'always', got %v", data["auto_download_photos"])
		}
		if data["upload_quality"].(string) != "high" {
			t.Errorf("Expected upload_quality to be 'high', got %v", data["upload_quality"])
		}
	})
}
