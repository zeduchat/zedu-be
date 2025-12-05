package test_file_management

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestPinnedFiles(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testpinned%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "PinnedUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_pinneduser%v", currUUID),
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

	t.Run("SetupFile", func(t *testing.T) {
		fileID = uploadTestFile(t, r, token, "pinned_doc.txt", "text/plain", []byte("pin me"))
	})

	var pinnedFileID string

	t.Run("PinFile_CaptureID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/file/%s/pin", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		pinnedFileID = data["pinned_file_id"].(string)
	})

	t.Run("PinFile_Duplicate", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/file/%s/pin", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		newPinnedFileID := data["pinned_file_id"].(string)

		if newPinnedFileID != pinnedFileID {
			t.Errorf("PinnedFile ID changed after duplicate pin. Original: %s, New: %s", pinnedFileID, newPinnedFileID)
		}
	})

	t.Run("GetPinnedFiles", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/favorites", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].([]interface{})
		if len(data) == 0 {
			t.Error("Expected at least one pinned file")
		}
		found := false
		for _, f := range data {
			item := f.(map[string]interface{})
			file := item["file"].(map[string]interface{})
			if file["id"].(string) == fileID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Pinned file not found in favorites list")
		}
	})

	t.Run("UnpinFile_HardDelete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s/pin", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		var count int64
		db.Postgresql.Unscoped().Model(&models.PinnedFile{}).Where("id = ?", pinnedFileID).Count(&count)
		if count != 0 {
			t.Errorf("PinnedFile should be permanently deleted (hard delete), but found %d records", count)
		}

		// Verify duplicate unpin returns 404
		req2, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s/pin", fileID), nil)
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		tests.AssertStatusCode(t, rr2.Code, http.StatusNotFound)
	})

	t.Run("GetPinnedFiles_Empty", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/favorites", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		resp := tests.ParseResponse(rr)
		data := resp["data"].([]interface{})

		if len(data) != 0 {
			t.Errorf("Expected empty favorites list, but got %d items", len(data))
		}
	})

	t.Run("UnpinFile_ByPinnedFileID", func(t *testing.T) {
		newFileID := uploadTestFile(t, r, token, "pinned_doc_2.txt", "text/plain", []byte("pin me 2"))
		reqPin, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/file/%s/pin", newFileID), nil)
		reqPin.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		rrPin := httptest.NewRecorder()
		r.ServeHTTP(rrPin, reqPin)
		tests.AssertStatusCode(t, rrPin.Code, http.StatusCreated)
		respPin := tests.ParseResponse(rrPin)
		dataPin := respPin["data"].(map[string]interface{})
		pID := dataPin["pinned_file_id"].(string)

		// unpin using PinnedFileID
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s/pin", pID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify deletion
		var count int64
		db.Postgresql.Unscoped().Model(&models.PinnedFile{}).Where("id = ?", pID).Count(&count)
		if count != 0 {
			t.Errorf("PinnedFile should be deleted when unpinning by PinnedFileID, but found %d records", count)
		}
	})
}
