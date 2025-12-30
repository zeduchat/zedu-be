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

func TestUploadQuality(t *testing.T) {
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

	// Create user-level media preferences with default upload quality
	mediaPreferences := models.MediaPreferences{
		ID:                    utility.GenerateUUID(),
		UserID:                userID,
		DeviceID:              nil,
		AutoDownloadPhotos:    "wifi_only",
		AutoDownloadAudio:     "wifi_only",
		AutoDownloadDocuments: "wifi_only",
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

	t.Run("Update Upload Quality to Standard", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := models.UpdateMediaPreferencesRequest{
			UploadQuality: "standard",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		data := response["data"].(map[string]any)
		if data["upload_quality"].(string) != "standard" {
			t.Errorf("Expected upload_quality to be 'standard', got %v", data["upload_quality"])
		}
	})

	t.Run("Update Upload Quality to High", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := models.UpdateMediaPreferencesRequest{
			UploadQuality: "high",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		data := response["data"].(map[string]any)
		if data["upload_quality"].(string) != "high" {
			t.Errorf("Expected upload_quality to be 'high', got %v", data["upload_quality"])
		}
	})

	t.Run("Update Upload Quality to Original", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

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
		data := response["data"].(map[string]any)
		if data["upload_quality"].(string) != "original" {
			t.Errorf("Expected upload_quality to be 'original', got %v", data["upload_quality"])
		}
	})

	t.Run("Invalid Upload Quality - Low", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := map[string]string{
			"upload_quality": "low",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	})

	t.Run("Invalid Upload Quality - Medium", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := map[string]string{
			"upload_quality": "medium",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	})

	t.Run("Invalid Upload Quality - Empty String", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		updateData := map[string]string{
			"upload_quality": "",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Empty string should be ignored (omitempty validation)
		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	})

	t.Run("Update Upload Quality for Device-Specific Preferences", func(t *testing.T) {
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

		// Update device-specific upload quality
		updateData := models.UpdateMediaPreferencesRequest{
			DeviceID:      &deviceID,
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
		data := response["data"].(map[string]any)
		if data["device_id"].(string) != deviceID {
			t.Errorf("Expected device_id to be %s, got %v", deviceID, data["device_id"])
		}
		if data["upload_quality"].(string) != "original" {
			t.Errorf("Expected upload_quality to be 'original', got %v", data["upload_quality"])
		}
	})

	t.Run("Get Upload Quality from User Preferences", func(t *testing.T) {
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
		data := response["data"].(map[string]any)

		// Verify upload_quality is present and valid
		uploadQuality, exists := data["upload_quality"]
		if !exists {
			t.Error("Expected upload_quality field in response")
		}

		uploadQualityStr := uploadQuality.(string)
		validValues := []string{"standard", "high", "original"}
		isValid := false
		for _, v := range validValues {
			if uploadQualityStr == v {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Errorf("Expected upload_quality to be one of %v, got %v", validValues, uploadQualityStr)
		}
	})

	t.Run("Get Upload Quality from Device Preferences with Fallback", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		// Create device preferences without upload_quality (should fallback to user-level)
		newDeviceID := utility.GenerateUUID()
		newDeviceUUID, _ := uuid.FromString(newDeviceID)
		newDevice := models.DeviceRegistry{
			ID:         newDeviceID,
			UserID:     userID,
			DeviceName: "New Device",
			DeviceType: "desktop",
			IsActive:   true,
		}
		db.Create(&newDevice)

		// Create device preferences with empty upload_quality
		devicePref := models.MediaPreferences{
			ID:                    utility.GenerateUUID(),
			UserID:                userID,
			DeviceID:              &newDeviceUUID,
			AutoDownloadPhotos:    "always",
			AutoDownloadAudio:     "always",
			AutoDownloadDocuments: "always",
			AutoDownloadVideos:    "always",
			UploadQuality:         "", // Empty, should fallback to user-level
		}
		db.Create(&devicePref)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/media-preferences?device_id=%s", newDeviceID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		data := response["data"].(map[string]any)

		// Should fallback to user-level upload_quality (standard)
		if data["upload_quality"].(string) != "standard" {
			t.Errorf("Expected upload_quality to fallback to 'standard' from user preferences, got %v", data["upload_quality"])
		}
	})

	t.Run("Upload Quality Persists After Update", func(t *testing.T) {
		router, authController := setup()
		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		// First update
		updateData1 := models.UpdateMediaPreferencesRequest{
			UploadQuality: "high",
		}
		body1, _ := json.Marshal(updateData1)

		req1, _ := http.NewRequest(http.MethodPut, "/api/v1/users/media-preferences", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp1 := httptest.NewRecorder()
		router.ServeHTTP(resp1, req1)
		tests.AssertStatusCode(t, resp1.Code, http.StatusOK)

		// Verify it was saved
		req2, _ := http.NewRequest(http.MethodGet, "/api/v1/users/media-preferences", nil)
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp2 := httptest.NewRecorder()
		router.ServeHTTP(resp2, req2)

		tests.AssertStatusCode(t, resp2.Code, http.StatusOK)
		response := tests.ParseResponse(resp2)
		data := response["data"].(map[string]any)
		if data["upload_quality"].(string) != "high" {
			t.Errorf("Expected upload_quality to persist as 'high', got %v", data["upload_quality"])
		}
	})
}
