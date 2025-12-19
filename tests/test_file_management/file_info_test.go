package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestGetFileInfo(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	// Setup test user
	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testfileinfo%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "FileInfoUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_fileinfo%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	tests.SignupUser(t, r, *authController, userSignUpData, false)
	token := tests.GetLoginToken(t, r, *authController, loginData)

	if token == "" {
		t.Fatal("Failed to get authentication token")
	}

	var user models.User
	db.Postgresql.Preload("Organisations").Where("email = ?", userSignUpData.Email).First(&user)
	if len(user.Organisations) == 0 {
		t.Fatal("User has no organisations")
	}

	var fileID string

	t.Run("UploadFile", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", "test_info_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("Test file for info endpoint"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-files", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("UploadFile failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		files, ok := resp["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data to be a list, got %T", resp["data"])
		}
		if len(files) == 0 {
			t.Fatal("Expected at least one file in response")
		}
		f := files[0].(map[string]interface{})
		fileID = f["id"].(string)
	})

	t.Run("GetFileInfo_Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/file/%s/info", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("GetFileInfo failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data to be an object, got %T", resp["data"])
		}

		// Verify response structure
		if _, exists := data["owner"]; !exists {
			t.Error("Expected 'owner' field in response")
		}
		if _, exists := data["date_uploaded"]; !exists {
			t.Error("Expected 'date_uploaded' field in response")
		}
		if _, exists := data["last_updated"]; !exists {
			t.Error("Expected 'last_updated' field in response")
		}
		if _, exists := data["shared_in"]; !exists {
			t.Error("Expected 'shared_in' field in response")
		}

		// Verify owner matches user
		owner := data["owner"].(string)
		if owner != userSignUpData.UserName {
			t.Errorf("Expected owner to be %s, got %s", userSignUpData.UserName, owner)
		}

		// Verify shared_in is an array
		sharedIn, ok := data["shared_in"].([]interface{})
		if !ok {
			t.Errorf("Expected shared_in to be an array, got %T", data["shared_in"])
		}

		// For a newly uploaded file not shared in channels, should be empty
		if len(sharedIn) != 0 {
			t.Logf("Note: shared_in has %d items: %v", len(sharedIn), sharedIn)
		}
	})

	t.Run("GetFileInfo_InvalidFileID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/file/invalid-uuid/info", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid UUID, got %d", http.StatusBadRequest, rr.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)

		if response["message"] != "Invalid file ID format" {
			t.Errorf("Expected error message about invalid format, got: %v", response["message"])
		}
	})

	t.Run("GetFileInfo_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/file/%s/info", nonExistentID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status %d for non-existent file, got %d", http.StatusNotFound, rr.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)

		if response["message"] != "File not found" {
			t.Errorf("Expected 'File not found' message, got: %v", response["message"])
		}
	})

	t.Run("GetFileInfo_Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/file/%s/info", fileID), nil)
		// No authorization header

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d for unauthorized request, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}
