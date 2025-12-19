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

	t.Run("VerifyFolderCount", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/folders", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folders := data["folders"].([]interface{})

		found := false
		for _, f := range folders {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == folderID {
				found = true
				if count, ok := folder["item_count"].(float64); !ok || count != 1 {
					t.Errorf("Expected item_count 1, got %v", folder["item_count"])
				}
				break
			}
		}
		if !found {
			t.Error("Folder not found")
		}

		// upload another file to this folder
		uploadTestFile(t, r, token, "second_file.txt", "text/plain", []byte("content2"), folderID)

		// check count is 2
		req2, _ := http.NewRequest(http.MethodGet, "/api/v1/files/folders", nil)
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		tests.AssertStatusCode(t, rr2.Code, http.StatusOK)
		resp2 := tests.ParseResponse(rr2)
		data2 := resp2["data"].(map[string]interface{})
		folders2 := data2["folders"].([]interface{})

		found2 := false
		for _, f := range folders2 {
			folder := f.(map[string]interface{})
			if folder["id"].(string) == folderID {
				found2 = true
				if count, ok := folder["item_count"].(float64); !ok || count != 2 {
					t.Errorf("Expected item_count 2, got %v", folder["item_count"])
				}
				break
			}
		}
		if !found2 {
			t.Error("Folder not found")
		}
	})

	t.Run("GetFoldersPagination", func(t *testing.T) {

		// creating 2 more folders with the existing one
		// to ensure enough for pagination
		for i := 0; i < 2; i++ {
			reqBody := map[string]string{
				"name": fmt.Sprintf("Pagination Folder %d", i),
			}
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(reqBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &b)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != http.StatusCreated {
				t.Fatalf("Failed to create folder for pagination test: %s", rr.Body.String())
			}
		}

		//get the folders with pagination
		u, _ := url.Parse("/api/v1/files/folders")
		q := u.Query()
		q.Set("page", "1")
		q.Set("limit", "1")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		folders := data["folders"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		if len(folders) != 1 {
			t.Errorf("Expected 1 folder in page 1, got %d", len(folders))
		}

		if pagination["current_page"].(float64) != 1 {
			t.Errorf("Expected current_page 1, got %v", pagination["current_page"])
		}

		totalItems := pagination["total_items"].(float64)
		if totalItems < 3 {
			t.Errorf("Expected at least 3 total items, got %v", totalItems)
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
			if pagination["total_items"] != 2.0 {
				t.Errorf("Expected total_items to be 2, got %v", pagination["total_items"])
			}
		}
	})

	t.Run("DeleteFile_Soft", func(t *testing.T) {
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

	t.Run("DeleteFile_Permanent", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s?permanent=true", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify hard delete in DB
		var file models.File
		err := db.Postgresql.Unscoped().Where("id = ?", fileID).First(&file).Error
		if err == nil {
			t.Errorf("File should be permanently deleted")
		}
	})

	var fileInFolderID string

	t.Run("DeleteFolder_Soft", func(t *testing.T) {

		// upload a file to the folder first
		fileInFolderID = uploadTestFile(t, r, token, "test_file_in_folder.txt", "text/plain", []byte("content"))

		// move it to the folder
		moveBody := []byte(fmt.Sprintf(`{"folder_id": "%s"}`, folderID))
		moveReq, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/%s/move", fileInFolderID), bytes.NewBuffer(moveBody))
		moveReq.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		moveReq.Header.Set("Content-Type", "application/json")
		moveRr := httptest.NewRecorder()
		r.ServeHTTP(moveRr, moveReq)
		tests.AssertStatusCode(t, moveRr.Code, http.StatusOK)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s", folderID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify folder was soft deleted
		var folder models.Folder
		err := db.Postgresql.Unscoped().Where("id = ?", folderID).First(&folder).Error
		if err != nil {
			t.Errorf("Folder should exist (soft deleted)")
		}
		if !folder.DeletedAt.Valid {
			t.Errorf("Folder should be soft deleted")
		}

		// verify file in folder was soft deleted
		var file models.File
		err = db.Postgresql.Unscoped().Where("id = ?", fileInFolderID).First(&file).Error
		if err != nil {
			t.Errorf("File in folder should exist")
		}
		if !file.DeletedAt.Valid {
			t.Errorf("File in folder should be soft deleted")
		}
	})

	t.Run("DeleteFolder_Permanent", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s?permanent=true", folderID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify folder was hard deleted
		var folder models.Folder
		err := db.Postgresql.Unscoped().Where("id = ?", folderID).First(&folder).Error
		if err == nil {
			t.Errorf("Folder should be permanently deleted")
		}

		// verify file in folder was hard deleted
		var file models.File
		err = db.Postgresql.Unscoped().Where("id = ?", fileInFolderID).First(&file).Error
		if err == nil {
			t.Errorf("File in folder should be permanently deleted")
		}
	})
}

func TestFileFilters(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testfilters%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "FilterUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_filteruser%v", currUUID),
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

	var imageFileID, docFileID, videoFileID string

	t.Run("SetupTestFiles", func(t *testing.T) {
		// JPEG magic number: FF D8 FF
		imageFileID = uploadTestFile(t, r, token, "test_image.jpg", "image/jpeg", []byte("\xFF\xD8\xFF\xE0"))

		// PDF magic number: %PDF-
		docFileID = uploadTestFile(t, r, token, "test_doc.pdf", "application/pdf", []byte("%PDF-1.4"))

		// MP4 magic number (ftyp box)
		videoFileID = uploadTestFile(t, r, token, "test_video.mp4", "video/mp4", []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00"))

		db.Postgresql.Model(&models.File{}).Where("id = ?", docFileID).
			Update("updated_at", time.Now().AddDate(0, 0, -10))
	})

	t.Run("FilterByFileCategory_Images", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "images")
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
			t.Error("Expected at least one image file")
		}

		for _, f := range files {
			file := f.(map[string]interface{})
			mimeType := file["mime_type"].(string)
			if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" {
				t.Errorf("Expected image mime type, got %s", mimeType)
			}
		}
	})

	t.Run("FilterByFileCategory_Documents", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "documents")
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
			t.Error("Expected at least one document file")
		}

		found := false
		for _, f := range files {
			file := f.(map[string]interface{})
			if file["id"].(string) == docFileID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find uploaded PDF in documents filter")
		}
	})

	t.Run("FilterByFileCategory_Videos", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "videos")
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
			t.Error("Expected at least one video file")
		}
	})

	t.Run("FilterByFileCategory_Invalid", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "invalid_category")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		resp := tests.ParseResponse(rr)
		if resp["message"] == "" {
			t.Error("Expected error message for invalid category")
		}
	})

	t.Run("FilterByDateModified_Today", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "today")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) < 2 {
			t.Error("Expected at least 2 files modified today")
		}

		for _, f := range files {
			file := f.(map[string]interface{})
			if file["id"].(string) == docFileID {
				t.Error("Document file modified 10 days ago should not appear in 'today' filter")
			}
		}
	})

	t.Run("FilterByDateModified_Last7Days", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "last_7_days")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) < 2 {
			t.Error("Expected at least 2 files in last 7 days")
		}
	})

	t.Run("FilterByDateModified_Last30Days", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "last_30_days")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) < 3 {
			t.Error("Expected all 3 files in last 30 days")
		}
	})

	t.Run("FilterByDateModified_ThisYear", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "this_year")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		if len(files) < 3 {
			t.Error("Expected all files to be modified this year")
		}
	})

	t.Run("FilterByDateModified_Invalid", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("date_modified", "invalid_date")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
		resp := tests.ParseResponse(rr)
		if resp["message"] == "" {
			t.Error("Expected error message for invalid date filter")
		}
	})

	t.Run("CombineFilters_CategoryAndDate", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("file_category", "images")
		q.Set("date_modified", "today")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		files := data["files"].([]interface{})

		for _, f := range files {
			file := f.(map[string]interface{})
			mimeType := file["mime_type"].(string)
			if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" {
				t.Errorf("Expected only image files, got %s", mimeType)
			}
		}
	})

	t.Run("CombineFilters_WithModeAndSearch", func(t *testing.T) {
		u, _ := url.Parse("/api/v1/files")
		q := u.Query()
		q.Set("mode", "mine")
		q.Set("file_category", "documents")
		q.Set("search", "test")
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)

		if resp["status"] != "success" {
			t.Error("Expected successful response with combined filters")
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		db.Postgresql.Unscoped().Delete(&models.File{}, "id IN ?", []string{imageFileID, docFileID, videoFileID})
	})
}

// Helper function to upload test files with specific mime types
func uploadTestFile(t *testing.T, r http.Handler, token, filename, mimeType string, content []byte, folderID ...string) string {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "files", filename)}
	h["Content-Type"] = []string{mimeType}
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)

	if len(folderID) > 0 && folderID[0] != "" {
		_ = writer.WriteField("folder_id", folderID[0])
	}
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Upload failed with status %d. Response: %s", rr.Code, rr.Body.String())
	}

	resp := tests.ParseResponse(rr)
	files := resp["data"].([]interface{})
	if len(files) == 0 {
		t.Fatal("No files returned from upload")
	}

	f := files[0].(map[string]interface{})
	fileID := f["id"].(string)

	return fileID
}
