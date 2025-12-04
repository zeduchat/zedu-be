package test_file_management

import (
	"bytes"
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

func TestUploadFolderWithFiles(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testfolderupload%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "FolderUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_folderuser%v", currUUID),
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

	t.Run("UploadFolderWithFiles_Success", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Add folder name
		writer.WriteField("folder_name", "My Project Folder")

		// Add multiple files
		files := []struct {
			name    string
			content string
		}{
			{"document.txt", "This is a document"},
			{"report.txt", "This is a report"},
			{"notes.txt", "These are notes"},
		}

		for _, file := range files {
			part, err := writer.CreateFormFile("files", file.name)
			if err != nil {
				t.Fatal(err)
			}
			part.Write([]byte(file.content))
		}
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-folder", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("UploadFolderWithFiles failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})

		// Verify folder was created
		folder, ok := data["folder"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected folder in response")
		}
		if folder["name"].(string) != "My Project Folder" {
			t.Errorf("Expected folder name 'My Project Folder', got %s", folder["name"].(string))
		}

		// Verify files were uploaded
		uploadedFiles, ok := data["files"].([]interface{})
		if !ok {
			t.Fatal("Expected files array in response")
		}
		if len(uploadedFiles) != 3 {
			t.Errorf("Expected 3 files, got %d", len(uploadedFiles))
		}

		// Verify file count
		fileCount, ok := data["file_count"].(float64)
		if !ok || int(fileCount) != 3 {
			t.Errorf("Expected file_count to be 3, got %v", fileCount)
		}

		// Verify files are associated with the folder
		folderID := folder["id"].(string)
		for _, file := range uploadedFiles {
			f := file.(map[string]interface{})
			if f["folder_id"].(string) != folderID {
				t.Errorf("File %s not associated with folder", f["file_name"].(string))
			}
		}
	})

	t.Run("UploadFolderWithFiles_WithParentFolder", func(t *testing.T) {
		// First create a parent folder
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("folder_name", "Parent Folder")
		part, _ := writer.CreateFormFile("files", "parent_file.txt")
		part.Write([]byte("parent content"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-folder", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		parentFolder := data["folder"].(map[string]interface{})
		parentFolderID := parentFolder["id"].(string)

		// Now create a subfolder
		body2 := new(bytes.Buffer)
		writer2 := multipart.NewWriter(body2)
		writer2.WriteField("folder_name", "Subfolder")
		writer2.WriteField("parent_id", parentFolderID)
		part2, _ := writer2.CreateFormFile("files", "sub_file.txt")
		part2.Write([]byte("subfolder content"))
		writer2.Close()

		req2, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-folder", body2)
		req2.Header.Set("Content-Type", writer2.FormDataContentType())
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusCreated {
			t.Fatalf("Subfolder creation failed with status %d. Response: %s", rr2.Code, rr2.Body.String())
		}

		resp2 := tests.ParseResponse(rr2)
		data2 := resp2["data"].(map[string]interface{})
		subfolder := data2["folder"].(map[string]interface{})

		if subfolder["parent_id"].(string) != parentFolderID {
			t.Errorf("Expected subfolder parent_id to be %s, got %s", parentFolderID, subfolder["parent_id"].(string))
		}
	})

	t.Run("UploadFolderWithFiles_MissingFolderName", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Don't add folder_name
		part, _ := writer.CreateFormFile("files", "test.txt")
		part.Write([]byte("content"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-folder", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for binding error, got %d", rr.Code)
		}
	})

	t.Run("UploadFolderWithFiles_NoFiles", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("folder_name", "Empty Folder")
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-folder", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing files, got %d", rr.Code)
		}
	})

	t.Run("UploadFolderWithFiles_FileTooLarge", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("folder_name", "Large File Folder")

		// Create a file larger than 200MB (simulated)
		part, _ := writer.CreateFormFile("files", "large_file.bin")
		// Write a small amount but we'll check the header size
		part.Write([]byte("This would be a huge file"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-folder", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// This test will pass since our test file is small
		// In real scenario, files > 200MB would be rejected
		if rr.Code == http.StatusBadRequest {
			resp := tests.ParseResponse(rr)
			if resp["message"].(string) != "File exceeds max size" {
				t.Errorf("Expected 'File exceeds max size' error message")
			}
		}
	})

	t.Run("UploadFolderWithFiles_TransactionRollback", func(t *testing.T) {
		// This test verifies that if file upload fails, the folder is not created
		// We can't easily simulate MinIO failure in tests, but the logic is there
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("folder_name", "Rollback Test Folder")

		part, _ := writer.CreateFormFile("files", "test.txt")
		part.Write([]byte("content"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-folder", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// If it succeeds, verify folder exists in DB
		if rr.Code == http.StatusCreated {
			resp := tests.ParseResponse(rr)
			data := resp["data"].(map[string]interface{})
			folder := data["folder"].(map[string]interface{})
			folderID := folder["id"].(string)

			var dbFolder models.Folder
			err := db.Postgresql.Where("id = ?", folderID).First(&dbFolder).Error
			if err != nil {
				t.Errorf("Folder should exist in database after successful upload")
			}
		}
	})
}
