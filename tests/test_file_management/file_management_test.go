package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestFileManagement(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testfm%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "FMUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_fmuser%v", currUUID),
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

	var folderID string
	var fileID string

	t.Run("CreateFolder", func(t *testing.T) {
		reqBody := map[string]string{
			"name": "Test Folder",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("CreateFolder failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folderID = data["id"].(string)
	})

	t.Run("UploadFile", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", "test_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("This is a test file content"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/upload-files?folder_id=%s", folderID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			fmt.Printf("UploadFile failed. Status: %d. Response: %s\n", rr.Code, rr.Body.String())
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

	t.Run("UploadDuplicateFile", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", "test_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("This is a test file content"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/upload-files?folder_id=%s", folderID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("UploadDuplicateFile failed with status %d. Response: %s", rr.Code, rr.Body.String())
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
		duplicateFileID := f["id"].(string)

		if duplicateFileID != fileID {
			t.Errorf("Expected duplicate upload to return existing file ID %s, got %s", fileID, duplicateFileID)
		}
	})

	t.Run("MoveFile", func(t *testing.T) {
		reqBody := map[string]string{
			"folder_id": folderID,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/%s/move", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify in DB
		var file models.File
		db.Postgresql.Where("id = ?", fileID).First(&file)
		if file.FolderID == nil || *file.FolderID != folderID {
			t.Errorf("File not moved to folder")
		}
	})

	t.Run("GetFiles_InFolder", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("folder_id", folderID)
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})
		if len(files) == 0 {
			t.Errorf("Expected files in folder")
		}

		pagination, ok := data["pagination"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected pagination object in response")
		} else {
			if pagination["current_page"] != 1.0 {
				t.Errorf("Expected current_page to be 1, got %v", pagination["current_page"])
			}
			if pagination["total_items"] == 0.0 {
				t.Errorf("Expected total_items to be > 0")
			}
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify soft delete in DB
		var file models.File
		db.Postgresql.Unscoped().Where("id = ?", fileID).First(&file)
		if file.DeletedAt.Time.IsZero() {
			t.Errorf("File should be soft deleted")
		}
	})

	t.Run("GetFiles_Trash", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("mode", "trash")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})
		found := false
		for _, f := range files {
			fMap := f.(map[string]interface{})
			if fMap["id"] == fileID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected deleted file in trash")
		}
	})

	t.Run("DeleteFolder", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s", folderID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify folder was deleted
		var folder models.Folder
		err := db.Postgresql.Where("id = ?", folderID).First(&folder).Error
		if err == nil {
			t.Errorf("Folder should be deleted")
		}
	})
}
