package test_file_management

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gofrsuuid "github.com/gofrs/uuid"
	stduuid "github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestDeleteFilePermissions(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	// owner user
	ownerSignup := newUserSignupData("owner-file")
	signupUser(t, authController, ownerSignup)
	ownerToken := loginUser(t, authController, models.LoginRequestModel{
		Email:    ownerSignup.Email,
		Password: ownerSignup.Password,
	})
	if ownerToken == "" {
		t.Fatal("failed to obtain owner token")
	}

	owner := getUserByEmail(t, db, ownerSignup.Email)
	if len(owner.Organisations) == 0 {
		t.Fatal("owner user missing organisation")
	}
	ownerOrgID := owner.Organisations[0].ID

	var ownerMembership models.OrgUserManagement
	if err := db.Postgresql.Where("user_id = ? AND organisation_id = ?", owner.ID, ownerOrgID).First(&ownerMembership).Error; err != nil {
		t.Fatalf("failed to fetch owner membership: %v", err)
	}
	ownerRoleID := ownerMembership.RoleID
	if ownerRoleID == "" && owner.OrgRoleID != nil {
		ownerRoleID = *owner.OrgRoleID
	}
	if ownerRoleID == "" {
		t.Fatal("owner role id not found")
	}

	createOwnerFile := func() string {
		return uploadTestFile(t, r, ownerToken)
	}

	// same org user without permission
	memberSignup := newUserSignupData("member-file")
	signupUser(t, authController, memberSignup)
	member := getUserByEmail(t, db, memberSignup.Email)
	assignUserToOrg(t, db, &member, ownerOrgID, ownerRoleID)
	memberToken := loginUser(t, authController, models.LoginRequestModel{
		Email:    memberSignup.Email,
		Password: memberSignup.Password,
	})

	// privileged user with can_delete_any_file
	privilegedSignup := newUserSignupData("privileged-file")
	signupUser(t, authController, privilegedSignup)
	privileged := getUserByEmail(t, db, privilegedSignup.Email)
	privilegedRoleID := createRoleWithPermission(t, db, ownerOrgID, models.PermissionList{
		CanDeleteAnyFile: true,
	})
	assignUserToOrg(t, db, &privileged, ownerOrgID, privilegedRoleID)
	privilegedToken := loginUser(t, authController, models.LoginRequestModel{
		Email:    privilegedSignup.Email,
		Password: privilegedSignup.Password,
	})

	// different org user
	otherOrgSignup := newUserSignupData("otherorg-file")
	signupUser(t, authController, otherOrgSignup)
	otherOrgToken := loginUser(t, authController, models.LoginRequestModel{
		Email:    otherOrgSignup.Email,
		Password: otherOrgSignup.Password,
	})

	t.Run("owner_can_delete_own_file", func(t *testing.T) {
		fileID := createOwnerFile()
		rr := performDeleteRequest(r, ownerToken, fileID, "")
		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		var file models.File
		if err := db.Postgresql.Unscoped().Where("id = ?", fileID).First(&file).Error; err != nil {
			t.Fatalf("expected file record for verification: %v", err)
		}
		if file.DeletedAt.Time.IsZero() {
			t.Errorf("expected file %s to be soft deleted", fileID)
		}
	})

	t.Run("same_org_without_permission_gets_forbidden", func(t *testing.T) {
		fileID := createOwnerFile()
		rr := performDeleteRequest(r, memberToken, fileID, "")
		tests.AssertStatusCode(t, rr.Code, http.StatusForbidden)

		var file models.File
		if err := db.Postgresql.Where("id = ?", fileID).First(&file).Error; err != nil {
			t.Fatalf("expected file to remain for failed delete: %v", err)
		}
		if !file.DeletedAt.Time.IsZero() {
			t.Errorf("file should remain undeleted when user lacks permission")
		}
	})

	t.Run("user_with_permission_can_delete_other_users_file", func(t *testing.T) {
		fileID := createOwnerFile()
		rr := performDeleteRequest(r, privilegedToken, fileID, "")
		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
	})

	t.Run("different_org_user_blocked", func(t *testing.T) {
		fileID := createOwnerFile()
		rr := performDeleteRequest(r, otherOrgToken, fileID, "")
		tests.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("nonexistent_file_returns_not_found", func(t *testing.T) {
		randomID := utility.GenerateUUID()
		rr := performDeleteRequest(r, ownerToken, randomID, "")
		tests.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})
}

func newUserSignupData(prefix string) models.CreateUserRequestModel {
	uid := stduuid.New().String()
	return models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%s_%s@qa.team", prefix, uid),
		FirstName:   "Test",
		LastName:    "User",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("%s_user_%s", prefix, uid),
	}
}

func getUserByEmail(t *testing.T, db *storage.Database, email string) models.User {
	var user models.User
	if err := db.Postgresql.Preload("Organisations").Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("failed to fetch user by email %s: %v", email, err)
	}
	return user
}

func assignUserToOrg(t *testing.T, db *storage.Database, user *models.User, orgID, roleID string) {
	orgUUID := gofrsuuid.FromStringOrNil(orgID)
	user.CurrentOrg = orgUUID
	if roleID != "" {
		user.OrgRoleID = &roleID
	}

	if err := db.Postgresql.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"current_org": orgUUID,
		"org_role_id": roleID,
	}).Error; err != nil {
		t.Fatalf("failed to update user org context: %v", err)
	}

	var membership models.OrgUserManagement
	err := db.Postgresql.Where("user_id = ? AND organisation_id = ?", user.ID, orgID).First(&membership).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			membership = models.OrgUserManagement{
				UserID:         user.ID,
				OrganisationID: orgID,
				RoleID:         roleID,
				Status:         "active",
			}
			if err := db.Postgresql.Create(&membership).Error; err != nil {
				t.Fatalf("failed to create org membership: %v", err)
			}
		} else {
			t.Fatalf("failed to fetch org membership: %v", err)
		}
	} else {
		membership.RoleID = roleID
		membership.Status = "active"
		if err := db.Postgresql.Save(&membership).Error; err != nil {
			t.Fatalf("failed to update org membership: %v", err)
		}
	}
}

func createRoleWithPermission(t *testing.T, db *storage.Database, orgID string, perms models.PermissionList) string {
	roleID := utility.GenerateUUID()
	orgIDCopy := orgID
	roleName := fmt.Sprintf("role-%s", stduuid.NewString()[:8])
	role := models.OrgRole{
		ID:             roleID,
		Name:           roleName,
		Description:    "File management test role",
		OrganisationID: &orgIDCopy,
		IsDefault:      false,
	}
	if err := role.CreateOrgRole(db.Postgresql); err != nil {
		t.Fatalf("failed to create org role: %v", err)
	}

	var permission models.Permission
	if err := db.Postgresql.Where("role_id = ?", roleID).First(&permission).Error; err != nil {
		t.Fatalf("failed to fetch role permission: %v", err)
	}
	permission.PermissionList = perms
	if err := db.Postgresql.Model(&models.Permission{}).Where("id = ?", permission.ID).Update("permission_list", permission.PermissionList).Error; err != nil {
		t.Fatalf("failed to update role permission: %v", err)
	}

	return roleID
}

func uploadTestFile(t *testing.T, r *gin.Engine, token string) string {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", fmt.Sprintf("delete-check-%s.txt", stduuid.New().String()))
	if err != nil {
		t.Fatalf("failed to create multipart part: %v", err)
	}
	part.Write([]byte("file delete permission test"))
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	tests.AssertStatusCode(t, rr.Code, http.StatusOK)

	resp := tests.ParseResponse(rr)
	files, ok := resp["data"].([]interface{})
	if !ok || len(files) == 0 {
		t.Fatalf("unexpected upload response: %+v", resp["data"])
	}
	first := files[0].(map[string]interface{})
	return first["id"].(string)
}

func performDeleteRequest(r *gin.Engine, token, fileID, threadID string) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/v1/files/file/%s", fileID)
	if threadID != "" {
		url = fmt.Sprintf("%s?thread_id=%s", url, threadID)
	}
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func signupUser(t *testing.T, authController *auth.Controller, signup models.CreateUserRequestModel) {
	authRouter := gin.Default()
	tests.SignupUser(t, authRouter, *authController, signup, false)
}

func loginUser(t *testing.T, authController *auth.Controller, login models.LoginRequestModel) string {
	authRouter := gin.Default()
	return tests.GetLoginToken(t, authRouter, *authController, login)
}
