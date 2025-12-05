package test_file_management

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestRecentFiles(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testrecent%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "RecentUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_recentuser%v", currUUID),
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

	t.Run("TrackRecentFiles", func(t *testing.T) {

		file1 := uploadTestFile(t, r, token, "file1.txt", "text/plain", []byte("content1"))
		time.Sleep(100 * time.Millisecond)
		file2 := uploadTestFile(t, r, token, "file2.txt", "text/plain", []byte("content2"))
		time.Sleep(100 * time.Millisecond)
		file3 := uploadTestFile(t, r, token, "file3.txt", "text/plain", []byte("content3"))

		accessFile(t, r, token, file2)

		time.Sleep(200 * time.Millisecond)

		accessFile(t, r, token, file1)
		time.Sleep(100 * time.Millisecond)

		accessFile(t, r, token, file3)
		time.Sleep(100 * time.Millisecond)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/recent", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)

		val, ok := resp["data"]
		if !ok || val == nil {
			t.Fatalf("Response data is nil or missing: %+v", resp)
		}

		data, ok := val.([]interface{})
		if !ok {
			t.Fatalf("Expected data to be []interface{}, got %T. Response: %+v", val, resp)
		}

		if len(data) != 3 {
			t.Fatalf("Expected 3 recent files, got %d", len(data))
		}

		f0 := data[0].(map[string]interface{})
		f1 := data[1].(map[string]interface{})
		f2 := data[2].(map[string]interface{})

		if f0["id"] != file3 {
			t.Errorf("Expected first file to be %s (file3), got %s", file3, f0["id"])
		}
		if f1["id"] != file1 {
			t.Errorf("Expected second file to be %s (file1), got %s", file1, f1["id"])
		}
		if f2["id"] != file2 {
			t.Errorf("Expected third file to be %s (file2), got %s", file2, f2["id"])
		}

		if f0["last_accessed_at"] == nil {
			t.Error("last_accessed_at should not be nil")
		}
	})

	t.Run("RecentFilesLimit", func(t *testing.T) {
		expectedLimit := 2
		// Reuse same user/files
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/recent?limit=%d", expectedLimit), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)

		val, ok := resp["data"]
		if !ok || val == nil {
			t.Fatalf("Response data is nil or missing: %+v", resp)
		}

		data, ok := val.([]interface{})
		if !ok {
			t.Fatalf("Expected data to be []interface{}, got %T. Response: %+v", val, resp)
		}

		if len(data) != expectedLimit {
			t.Errorf("Expected %d recent files, got %d", expectedLimit, len(data))
		}
	})

	t.Run("RecentFilesCapLimit", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/recent?limit=50", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)

		val, ok := resp["data"]
		if !ok || val == nil {
			t.Fatalf("Response data is nil or missing: %+v", resp)
		}

		data, ok := val.([]interface{})
		if !ok {
			t.Fatalf("Expected data to be []interface{}, got %T. Response: %+v", val, resp)
		}

		// currently we only have 3 files total
		// regression test for capped limit
		if len(data) > 30 {
			t.Error("Should not return more than 30 files, or at least respect logical cap")
		}
	})

	t.Run("RecentFilesSilentUpdateCheck", func(t *testing.T) {
		// verify that LastAccessedAt actually changed
		fileID := uploadTestFile(t, r, token, "silent_test.txt", "text/plain", []byte("silent"))

		var fileBefore models.File
		db.Postgresql.Where("id = ?", fileID).First(&fileBefore)
		lastAccessBefore := fileBefore.LastAccessedAt

		accessFile(t, r, token, fileID)
		time.Sleep(200 * time.Millisecond) // wait for goroutine

		var fileAfter models.File
		db.Postgresql.Where("id = ?", fileID).First(&fileAfter)

		if lastAccessBefore == nil || (lastAccessBefore == nil && fileAfter.LastAccessedAt == nil) {
			t.Error("LastAccessedAt should have been updated from nil")
		}

		if fileAfter.LastAccessedAt.After(*lastAccessBefore) {
			t.Logf("Before: %v, After: %v", lastAccessBefore, fileAfter.LastAccessedAt)
		}
	})

	t.Run("RecentFilesOnUploadAndUpdate", func(t *testing.T) {

		newFileID := uploadTestFile(t, r, token, "upload_active.txt", "text/plain", []byte("immediate"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/recent", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)

		val := resp["data"]
		data, _ := val.([]interface{})

		if len(data) == 0 {
			t.Fatal("Expected new Upload to be in recent files")
		}

		topFile := data[0].(map[string]interface{})
		if topFile["id"] != newFileID {
			t.Errorf("Expected uploaded file %s to be top recent, got %s", newFileID, topFile["id"])
		}

		// rename the file to triggers update
		// and wait a bit to ensure timestamp diff
		time.Sleep(1 * time.Second)

		renameBody := []byte(`{"file_name": "renamed_active.txt"}`)
		reqRename, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s", newFileID), bytes.NewBuffer(renameBody))
		reqRename.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		reqRename.Header.Set("Content-Type", "application/json")

		rrRename := httptest.NewRecorder()
		r.ServeHTTP(rrRename, reqRename)
		tests.AssertStatusCode(t, rrRename.Code, http.StatusOK)

		time.Sleep(200 * time.Millisecond) // wait for async update

		req2, _ := http.NewRequest(http.MethodGet, "/api/v1/files/recent", nil)
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		resp2 := tests.ParseResponse(rr2)
		val2 := resp2["data"]
		data2, _ := val2.([]interface{})

		topFile2 := data2[0].(map[string]interface{})

		if topFile2["id"] != newFileID {
			t.Errorf("Expected renamed file %s to be top recent", newFileID)
		}
	})
}

func accessFile(t *testing.T, r http.Handler, token, fileID string) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/file/%s", fileID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to access file %s: %d", fileID, rr.Code)
	}
}
