package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

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
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
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

	var user models.User
	db.Postgresql.Preload("Organisations").Where("email = ?", userSignUpData.Email).First(&user)

	if len(user.Organisations) == 0 {
		t.Fatal("User has no organisations")
	}

	// Create a test file entry
	testFile := models.File{
		ID:             utility.GenerateUUID(),
		FileName:       "original_file.txt",
		FileType:       "txt",
		MimeType:       "text/plain",
		FileLink:       "https://example.com/test-file.txt",
		OrganisationID: user.Organisations[0].ID,
		UserID:         user.ID,
	}

	var org models.Organisation
	// assuming user is owner of an org created during signup
	db.Postgresql.Where("owner_id = ?", user.ID).First(&org)
	testFile.OrganisationID = org.ID

	err := testFile.CreateFileRecord(db.Postgresql)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Cleanup after test
	defer func() {
		db.Postgresql.Where("id = ?", testFile.ID).Delete(&models.File{})
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
		if response["message"] != "Validation failed" {
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

	t.Run("TestUpdateFolderName", func(t *testing.T) {
		r, _, authController, _ := SetupFileManagementTestRouter()

		// Create a test folder using the API
		createFolderData := models.CreateFolderParams{
			Name: "original_folder",
		}

		createPath := "/api/v1/files/folders"
		createURI := url.URL{Path: createPath}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createFolderData)
		req, err := http.NewRequest(http.MethodPost, createURI.String(), &b)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		response := tests.ParseResponse(rr)
		data := response["data"].(map[string]interface{})
		folderID := data["id"].(string)

		t.Run("UpdateFolderName_Success", func(t *testing.T) {
			renameData := models.RenameFolderRequest{
				FolderName: "renamed_folder",
			}

			updatePath := fmt.Sprintf("/api/v1/files/folders/%s", folderID)
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

			tests.AssertStatusCode(t, rr.Code, http.StatusOK)

			response := tests.ParseResponse(rr)
			tests.AssertResponseMessage(t, response["message"].(string), "Folder name updated successfully")

			// Verify DB update
			var updatedFolder models.Folder
			db.Postgresql.Where("id = ?", folderID).First(&updatedFolder)
			if updatedFolder.Name != "renamed_folder" {
				t.Errorf("Expected folder name 'renamed_folder', got '%s'", updatedFolder.Name)
			}
		})

		t.Run("UpdateFolderName_EmptyName", func(t *testing.T) {
			renameData := models.RenameFolderRequest{
				FolderName: "",
			}

			updatePath := fmt.Sprintf("/api/v1/files/folders/%s", folderID)
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

		t.Run("UpdateFolderName_Unauthorized", func(t *testing.T) {
			// create another user
			otherUserUUID := uuid.New().String()
			otherUser := models.CreateUserRequestModel{
				Email:       fmt.Sprintf("otheruser%v@qa.team", otherUserUUID),
				FirstName:   "Other",
				LastName:    "User",
				PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
				Password:    "password123",
				UserName:    fmt.Sprintf("other_user%v", otherUserUUID),
			}

			tests.SignupUser(t, r, *authController, otherUser, false)

			loginData := models.LoginRequestModel{
				Email:    otherUser.Email,
				Password: otherUser.Password,
			}
			otherToken := tests.GetLoginToken(t, r, *authController, loginData)

			renameData := models.RenameFolderRequest{
				FolderName: "hacked_folder",
			}

			updatePath := fmt.Sprintf("/api/v1/files/folders/%s", folderID)
			updateURI := url.URL{Path: updatePath}

			var b bytes.Buffer
			json.NewEncoder(&b).Encode(renameData)
			req, err := http.NewRequest(http.MethodPut, updateURI.String(), &b)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", otherToken))

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		})

		t.Run("UpdateFolderName_NotFound", func(t *testing.T) {
			renameData := models.RenameFolderRequest{
				FolderName: "new_name",
			}

			nonExistentID := utility.GenerateUUID()
			updatePath := fmt.Sprintf("/api/v1/files/folders/%s", nonExistentID)
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

			if rr.Code == http.StatusOK {
				t.Errorf("Expected failure for non-existent folder, got 200")
			}
		})
	})
}
