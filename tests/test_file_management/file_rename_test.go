package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/fileManagement"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateFileName(t *testing.T) {
	logger := tests.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	// Setup test user
	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testfileuser%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "FileUser",
		PhoneNumber: "1234567890",
		Password:    "password123",
		UserName:    fmt.Sprintf("test_fileuser%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	// Create user and get token
	r := gin.Default()
	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}
	tests.SignupUser(t, r, authController, userSignUpData, false)
	token := tests.GetLoginToken(t, r, authController, loginData)

	if token == "" {
		t.Fatal("Failed to get authentication token")
	}

	// Create a test file entry
	testFile := models.UploadedFileResponse{
		ID:       utility.GenerateUUID(),
		FileName: "original_file.txt",
		FileType: "txt",
		MimeType: "text/plain",
		FileLink: "https://example.com/test-file.txt",
	}

	err := testFile.CreateFileRecord(db.Postgresql)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Cleanup after test
	defer func() {
		db.Postgresql.Where("id = ?", testFile.ID).Delete(&models.UploadedFileResponse{})
	}()

	fileController := fileManagement.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	t.Run("UpdateFileName_Success", func(t *testing.T) {
		renameData := models.RenameFileRequest{
			FileName: "renamed_file.txt",
		}

		updatePath := fmt.Sprintf("/api/v1/files/file/%s", testFile.ID)
		updateURI := url.URL{Path: updatePath}

		fileUrl := r.Group("/api/v1/files", middleware.Authorize(db.Postgresql))
		{
			fileUrl.PUT("/file/:id", fileController.UpdateFileName)
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameData)
		req, err := http.NewRequest(http.MethodPut, updateURI.String(), &b)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		response := tests.ParseResponse(rr)
		tests.AssertResponseMessage(t, response["message"].(string), "File name updated successfully")

		// Verify the file was actually updated in the database
		data := response["data"].(map[string]interface{})
		updatedFileName := data["file_name"].(string)
		if updatedFileName != "renamed_file.txt" {
			t.Errorf("Expected file name to be 'renamed_file.txt', got '%s'", updatedFileName)
		}
	})

	t.Run("UpdateFileName_EmptyName", func(t *testing.T) {
		renameData := models.RenameFileRequest{
			FileName: "",
		}

		updatePath := fmt.Sprintf("/api/v1/files/file/%s", testFile.ID)
		updateURI := url.URL{Path: updatePath}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameData)
		req, err := http.NewRequest(http.MethodPut, updateURI.String(), &b)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})

	t.Run("UpdateFileName_InvalidCharacters", func(t *testing.T) {
		renameData := models.RenameFileRequest{
			FileName: "../../../etc/passwd",
		}

		updatePath := fmt.Sprintf("/api/v1/files/file/%s", testFile.ID)
		updateURI := url.URL{Path: updatePath}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameData)
		req, err := http.NewRequest(http.MethodPut, updateURI.String(), &b)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

		response := tests.ParseResponse(rr)
		if response["message"] != "Failed to update file name" {
			t.Errorf("Expected error message about invalid characters")
		}
	})

	t.Run("UpdateFileName_FileNotFound", func(t *testing.T) {
		renameData := models.RenameFileRequest{
			FileName: "new_name.txt",
		}

		nonExistentID := utility.GenerateUUID()
		updatePath := fmt.Sprintf("/api/v1/files/file/%s", nonExistentID)
		updateURI := url.URL{Path: updatePath}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameData)
		req, err := http.NewRequest(http.MethodPut, updateURI.String(), &b)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})

	t.Run("UpdateFileName_TooLong", func(t *testing.T) {
		// Create a filename longer than 255 characters
		longName := ""
		for i := 0; i < 260; i++ {
			longName += "a"
		}

		renameData := models.RenameFileRequest{
			FileName: longName,
		}

		updatePath := fmt.Sprintf("/api/v1/files/file/%s", testFile.ID)
		updateURI := url.URL{Path: updatePath}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameData)
		req, err := http.NewRequest(http.MethodPut, updateURI.String(), &b)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// Validation errors return 422, service errors return 400
		if rr.Code != http.StatusBadRequest && rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 400 or 422, got %d", rr.Code)
		}
	})

	t.Run("UpdateFileName_Unauthorized", func(t *testing.T) {
		renameData := models.RenameFileRequest{
			FileName: "unauthorized.txt",
		}

		updatePath := fmt.Sprintf("/api/v1/files/file/%s", testFile.ID)
		updateURI := url.URL{Path: updatePath}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameData)
		req, err := http.NewRequest(http.MethodPut, updateURI.String(), &b)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})
}
