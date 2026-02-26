package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestEditPermissionEnforcement(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	// Setup first user (file owner)
	currUUID := uuid.New().String()
	ownerSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("fileowner%v@qa.team", currUUID),
		FirstName:   "File",
		LastName:    "Owner",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("file_owner%v", currUUID),
	}

	// Setup second user (will receive share)
	secondUUID := uuid.New().String()
	recipientSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("filerecipient%v@qa.team", secondUUID),
		FirstName:   "File",
		LastName:    "Recipient",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()+1),
		Password:    "password123",
		UserName:    fmt.Sprintf("file_recipient%v", secondUUID),
	}

	// Create both users
	tests.SignupUser(t, r, *authController, ownerSignUpData, false)
	tests.SignupUser(t, r, *authController, recipientSignUpData, false)

	// Get both users
	var owner, recipient models.User
	db.Postgresql.Preload("Organisations").Where("email = ?", ownerSignUpData.Email).First(&owner)
	db.Postgresql.Preload("Organisations").Where("email = ?", recipientSignUpData.Email).First(&recipient)

	if len(owner.Organisations) == 0 || len(recipient.Organisations) == 0 {
		t.Fatal("Users have no organisations")
	}

	// Add recipient to owner's organization
	var org models.Organisation
	db.Postgresql.Where("owner_id = ?", owner.ID).First(&org)

	// Get owner's role ID to use for the recipient
	var ownerMembership models.OrgUserManagement
	if err := db.Postgresql.Where("user_id = ? AND organisation_id = ?", owner.ID, org.ID).First(&ownerMembership).Error; err != nil {
		t.Fatalf("Failed to get owner membership: %v", err)
	}

	// Update recipient's current_org to owner's org
	if err := db.Postgresql.Model(&models.User{}).Where("id = ?", recipient.ID).Update("current_org", org.ID).Error; err != nil {
		t.Fatalf("Failed to update recipient's current_org: %v", err)
	}

	orgMember := models.OrgUserManagement{
		UserID:         recipient.ID,
		OrganisationID: org.ID,
		RoleID:         ownerMembership.RoleID,
		Status:         "active",
		IsDeactivated:  false,
	}
	if err := db.Postgresql.Create(&orgMember).Error; err != nil {
		t.Fatalf("Failed to create org membership: %v", err)
	}

	// Now login AFTER updating org so token has correct org_id
	ownerLoginData := models.LoginRequestModel{
		Email:    ownerSignUpData.Email,
		Password: ownerSignUpData.Password,
	}
	recipientLoginData := models.LoginRequestModel{
		Email:    recipientSignUpData.Email,
		Password: recipientSignUpData.Password,
	}

	ownerToken := tests.GetLoginToken(t, r, *authController, ownerLoginData)
	recipientToken := tests.GetLoginToken(t, r, *authController, recipientLoginData)

	if ownerToken == "" || recipientToken == "" {
		t.Fatal("Failed to get authentication tokens")
	}

	var fileID string
	var viewShareID, editShareID string
	var thirdUserModel models.User // Declare at function scope for cleanup access

	// Test 1: Create a test file
	t.Run("SetupTestFile", func(t *testing.T) {
		fileID = uploadTestFile(t, r, ownerToken, "test_permission.txt", "text/plain", []byte("test content"))
		if fileID == "" {
			t.Fatal("Failed to create test file")
		}
	})

	// Test 2: Create share with VIEW permission
	t.Run("CreateViewShare", func(t *testing.T) {
		shareReq := map[string]interface{}{
			"access_type":     "private",
			"permission_type": "view",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(shareReq)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/share", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ownerToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		viewShareID = data["file_share_id"].(string)

		assert.Equal(t, "view", data["permission_type"], "Expected view permission")
	})

	// Test 3: User with VIEW permission cannot rename file
	t.Run("ViewPermission_CannotRename", func(t *testing.T) {
		renameReq := map[string]string{
			"file_name": "attempted_rename.txt",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameReq)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", recipientToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusForbidden)

		resp := tests.ParseResponse(rr)
		message := resp["message"].(string)
		assert.Contains(t, strings.ToLower(message), "forbidden", "Expected permission denied message")
	})

	// Test 4: User with VIEW permission cannot delete file
	t.Run("ViewPermission_CannotDelete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/file/%s", fileID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", recipientToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusForbidden)

		resp := tests.ParseResponse(rr)
		message := resp["message"].(string)
		assert.Contains(t, strings.ToLower(message), "forbidden", "Expected permission denied message")
	})

	// Test 5: Create share with EDIT permission
	t.Run("CreateEditShare", func(t *testing.T) {
		shareReq := map[string]interface{}{
			"access_type":     "private",
			"permission_type": "edit",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(shareReq)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/files/%s/share", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ownerToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		editShareID = data["file_share_id"].(string)

		assert.Equal(t, "edit", data["permission_type"], "Expected edit permission")
	})

	// Test 6: User with EDIT permission CAN rename file
	t.Run("EditPermission_CanRename", func(t *testing.T) {
		renameReq := map[string]string{
			"file_name": "edited_renamed.txt",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameReq)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", recipientToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "edited_renamed.txt", data["file_name"], "File should be renamed")
	})

	// Test 7: File owner can always edit (even without shares)
	t.Run("Owner_CanAlwaysEdit", func(t *testing.T) {
		renameReq := map[string]string{
			"file_name": "owner_renamed.txt",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameReq)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ownerToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		resp := tests.ParseResponse(rr)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "owner_renamed.txt", data["file_name"], "Owner should be able to rename")
	})

	// Test 8: Non-owner without any share cannot edit
	t.Run("NoShare_CannotEdit", func(t *testing.T) {
		// Create a third user with no shares
		thirdUUID := uuid.New().String()
		thirdUser := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("third%v@qa.team", thirdUUID),
			FirstName:   "Third",
			LastName:    "User",
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()+2),
			Password:    "password123",
			UserName:    fmt.Sprintf("third_user%v", thirdUUID),
		}

		tests.SignupUser(t, r, *authController, thirdUser, false)
		thirdLoginData := models.LoginRequestModel{
			Email:    thirdUser.Email,
			Password: thirdUser.Password,
		}
		thirdToken := tests.GetLoginToken(t, r, *authController, thirdLoginData)

		// Add third user to org
		db.Postgresql.Where("email = ?", thirdUser.Email).First(&thirdUserModel)

		// Get owner's role ID to use for the third user
		var ownerMembership models.OrgUserManagement
		if err := db.Postgresql.Where("user_id = ? AND organisation_id = ?", owner.ID, org.ID).First(&ownerMembership).Error; err != nil {
			t.Fatalf("Failed to get owner membership: %v", err)
		}

		// Update third user's current_org to owner's org
		if err := db.Postgresql.Model(&models.User{}).Where("id = ?", thirdUserModel.ID).Update("current_org", org.ID).Error; err != nil {
			t.Fatalf("Failed to update third user's current_org: %v", err)
		}

		thirdOrgMember := models.OrgUserManagement{
			UserID:         thirdUserModel.ID,
			OrganisationID: org.ID,
			RoleID:         ownerMembership.RoleID,
			Status:         "active",
			IsDeactivated:  false,
		}
		if err := db.Postgresql.Create(&thirdOrgMember).Error; err != nil {
			t.Fatalf("Failed to create org membership: %v", err)
		}

		renameReq := map[string]string{
			"file_name": "third_user_rename.txt",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(renameReq)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/files/file/%s", fileID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", thirdToken))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	// Cleanup
	db.Postgresql.Unscoped().Where("id = ?", viewShareID).Delete(&models.FileShare{})
	db.Postgresql.Unscoped().Where("id = ?", editShareID).Delete(&models.FileShare{})
	db.Postgresql.Unscoped().Where("id = ?", fileID).Delete(&models.File{})
	db.Postgresql.Where("user_id = ? AND organisation_id = ?", recipient.ID, org.ID).Delete(&models.OrgUserManagement{})
	db.Postgresql.Where("user_id = ? AND organisation_id = ?", thirdUserModel.ID, org.ID).Delete(&models.OrgUserManagement{})
}
