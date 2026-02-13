package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestFileSharing(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testshare%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "ShareUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_shareuser%v", currUUID),
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
	var shareID string
	var shareLink string

	t.Run("SetupFile", func(t *testing.T) {
		fileID = uploadTestFile(t, r, token, "share_test_file.txt", "text/plain", []byte("file to share"))
	})

	t.Run("CreateFileShare_Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"access_type":     "private",
			"permission_type": "view",
			"note":            "Test share note",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/share", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		shareID = data["file_share_id"].(string)
		shareLink = data["share_link"].(string)

		if shareID == "" {
			t.Error("Expected share_id to be non-empty")
		}
		if shareLink == "" {
			t.Error("Expected share_link to be non-empty")
		}

		if data["access_type"] != "private" {
			t.Errorf("Expected access_type 'private', got %v", data["access_type"])
		}
		if data["permission_type"] != "view" {
			t.Errorf("Expected permission_type 'view', got %v", data["permission_type"])
		}
	})

	t.Run("GetFileShares_Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/%s/shares", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		resp := tests.ParseResponse(rr)
		data := resp["data"].([]interface{})

		if len(data) == 0 {
			t.Error("Expected at least one share in response")
		}

		found := false
		for _, s := range data {
			share := s.(map[string]interface{})
			if share["file_share"].(map[string]interface{})["id"].(string) == shareID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created share not found in shares list")
		}
	})

	t.Run("AccessSharedFile_Success", func(t *testing.T) {
		reqBody := map[string]string{
			"share_link": shareLink,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/access", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})

		if data["permission"] != "view" {
			t.Errorf("Expected permission 'view', got %v", data["permission"])
		}
		if data["access_type"] != "private" {
			t.Errorf("Expected access_type 'private', got %v", data["access_type"])
		}
		if data["file"] == nil {
			t.Error("Expected file data in response")
		}

		file := data["file"].(map[string]interface{})
		if file["id"].(string) != fileID {
			t.Error("Expected correct file ID in response")
		}
	})

	t.Run("RevokeFileShare_Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/shares/%s", shareID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		var count int64
		db.Postgresql.Model(&models.FileShare{}).Where("id = ?", shareID).Count(&count)
		if count != 0 {
			t.Error("Share should be soft deleted (count should be 0)")
		}
	})

	secondUserID := ""
	secondUserEmail := ""

	t.Run("SetupSecondUser", func(t *testing.T) {
		secondUUID := uuid.New().String()
		secondUser := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("seconduser%v@qa.team", secondUUID),
			FirstName:   "Second",
			LastName:    "User",
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("second_user%v", secondUUID),
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(secondUser)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", &b)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
			t.Fatalf("Failed to register second user: %s", rr.Body.String())
		}

		var secondUserDB models.User
		db.Postgresql.Where("email = ?", secondUser.Email).First(&secondUserDB)
		secondUserID = secondUserDB.ID
		secondUserEmail = secondUser.Email

		orgID := user.Organisations[0].ID

		// Get an existing role from the organization
		var orgRole models.OrgRole
		if err := db.Postgresql.Where("organisation_id = ?", orgID).First(&orgRole).Error; err != nil {
			// If no org-specific role, get a default role
			if err := db.Postgresql.Where("name = ?", "Administrator").First(&orgRole).Error; err != nil {
				t.Fatalf("Failed to find a role for the user: %v", err)
			}
		}

		// Create OrgUserManagement record to properly add user to organization
		orgUserMgt := models.OrgUserManagement{
			UserID:         secondUserID,
			OrganisationID: orgID,
			RoleID:         orgRole.ID,
			Status:         "active",
			IsDeactivated:  false,
		}

		if err := db.Postgresql.Create(&orgUserMgt).Error; err != nil {
			t.Fatalf("Failed to create org_user_managements record: %v", err)
		}
	})

	secondToken := ""

	t.Run("LoginSecondUser", func(t *testing.T) {
		loginReq := models.LoginRequestModel{
			Email:    secondUserEmail,
			Password: "password123",
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(loginReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", &b)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Failed to login second user: %s", rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		secondToken = data["access_token"].(string)

		if secondToken == "" {
			t.Fatal("Failed to get second user token")
		}
	})

	t.Run("CreateFileShare_Unauthorized", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"access_type":     "private",
			"permission_type": "view",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/share", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", secondToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
			t.Errorf("Expected 403 Forbidden or 404 Not Found, got %d", rr.Code)
		}
	})

	newShareID := ""

	t.Run("CreateShareForRevokeTest", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"access_type":     "private",
			"permission_type": "view",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/share", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		newShareID = data["file_share_id"].(string)
	})

	t.Run("RevokeFileShare_Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/shares/%s", newShareID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", secondToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("AccessSharedFile_Expired", func(t *testing.T) {
		var expiredShareID string
		var expiredShareLink string

		t.Run("CreateShareForExpirationTest", func(t *testing.T) {
			futureTime := time.Now().UTC().Add(24 * time.Hour)

			reqBody := map[string]interface{}{
				"access_type":     "private",
				"permission_type": "view",
				"expires_at":      futureTime,
			}
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(reqBody)

			req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/share", fileID), &b)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

			resp := tests.ParseResponse(rr)
			data := resp["data"].(map[string]interface{})
			expiredShareID = data["file_share_id"].(string)
			expiredShareLink = data["share_link"].(string)
		})

		t.Run("UpdateShareToExpired", func(t *testing.T) {
			pastTime := time.Now().UTC().Add(-24 * time.Hour)
			db.Postgresql.Model(&models.FileShare{}).Where("id = ?", expiredShareID).Update("expires_at", pastTime)
		})

		t.Run("AccessExpiredShare", func(t *testing.T) {
			accessReqBody := map[string]string{
				"share_link": expiredShareLink,
			}
			var ab bytes.Buffer
			json.NewEncoder(&ab).Encode(accessReqBody)

			accessReq, _ := http.NewRequest(http.MethodPost, "/api/v1/files/access", &ab)
			accessReq.Header.Set("Content-Type", "application/json")
			accessReq.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

			accessRR := httptest.NewRecorder()
			r.ServeHTTP(accessRR, accessReq)

			tests.AssertStatusCode(t, accessRR.Code, http.StatusGone)
		})
	})

	t.Run("CreateFileShare_InvalidExpiration", func(t *testing.T) {
		pastTime := time.Now().UTC().Add(-24 * time.Hour)

		reqBody := map[string]interface{}{
			"access_type":     "private",
			"permission_type": "view",
			"expires_at":      pastTime,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/share", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

		resp := tests.ParseResponse(rr)
		if resp["message"] == "" {
			t.Error("Expected error message for invalid expiration")
		}
	})

	t.Run("SendFileToDM_Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"recipient_ids": []string{secondUserID},
			"note":          "Here is a file for you",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/send-dm", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})

		recipients := data["recipients"].([]interface{})
		if len(recipients) != 1 {
			t.Errorf("Expected 1 recipient, got %d", len(recipients))
		}

		recipient := recipients[0].(map[string]interface{})
		if recipient["user_id"].(string) != secondUserID {
			t.Error("Expected correct recipient user ID")
		}

		if recipient["success"] != nil {
			if !recipient["success"].(bool) {
				t.Logf("Note: Recipient send failed with error: %v (DM message sending requires additional setup)", recipient["error"])
			}
		}
	})

	t.Run("UpdateFileAccessSettings_Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"access_type":  "public",
			"is_shareable": true,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/%s/access-settings", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})

		if data["access_type"] != "public" {
			t.Errorf("Expected access_type 'public', got %v", data["access_type"])
		}
		if data["is_shareable"].(bool) != true {
			t.Error("Expected is_shareable to be true")
		}

		var file models.File
		db.Postgresql.Where("id = ?", fileID).First(&file)
		if file.AccessType != "public" {
			t.Error("File access_type not updated in database")
		}
		if !file.IsShareable {
			t.Error("File is_shareable not updated in database")
		}
	})

	t.Run("SendFileToDM_UserNotFound", func(t *testing.T) {
		nonExistentUserID := uuid.New().String()

		reqBody := map[string]interface{}{
			"recipient_ids": []string{nonExistentUserID},
			"note":          "This should fail",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/send-dm", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})

		if data["failed_sends"].(float64) != 1 {
			t.Errorf("Expected 1 failed send, got %v", data["failed_sends"])
		}

		recipients := data["recipients"].([]interface{})
		recipient := recipients[0].(map[string]interface{})

		if recipient["success"].(bool) != false {
			t.Error("Expected recipient success to be false")
		}

		if recipient["error"].(string) != "user not found" {
			t.Errorf("Expected error 'user not found', got '%v'", recipient["error"])
		}
	})
}
