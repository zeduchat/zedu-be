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
		pinnedFileID = data["id"].(string)
	})

	t.Run("PinFile_Duplicate", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/file/%s/pin", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		newPinnedFileID := data["id"].(string)

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
			file := f.(map[string]interface{})
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
}
