package test_file_management

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

// TestEditPermissionScope covers #2511: CheckFileEditPermission (rename, delete, move)
// only grants edit rights to the owner, to holders of can_delete_any_file in the
// file's own organisation, and to users an edit share was actually delivered to.
func TestEditPermissionScope(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	var userRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", "User").First(&userRole).Error; err != nil {
		t.Fatalf("failed to find User role: %v", err)
	}

	type testUser struct {
		token, id, orgID string
	}
	// newUser signs a user up; with joinOrg set, the user joins that organisation with
	// roleID (the User role when empty) and logs in with it as the token organisation.
	newUser := func(tag, joinOrg, roleID string) testUser {
		currUUID := uuid.New().String()
		ud := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("editscope_%s_%v@qa.team", tag, currUUID),
			FirstName:   "Edit",
			LastName:    "Scope" + tag,
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("editscope_%s_%v", tag, currUUID),
		}
		tests.SignupUser(t, r, *authController, ud, false)
		var u models.User
		db.Postgresql.Preload("Organisations").Where("email = ?", ud.Email).First(&u)
		if len(u.Organisations) == 0 {
			t.Fatalf("user %s has no organisation", tag)
		}
		orgID := u.Organisations[0].ID
		if joinOrg != "" {
			if roleID == "" {
				roleID = userRole.ID
			}
			if err := db.Postgresql.Create(&models.OrgUserManagement{
				UserID: u.ID, OrganisationID: joinOrg, RoleID: roleID, Status: "active",
			}).Error; err != nil {
				t.Fatalf("failed to add membership: %v", err)
			}
			db.Postgresql.Model(&models.User{}).Where("id = ?", u.ID).Update("current_org", joinOrg)
			orgID = joinOrg
		}
		token := tests.GetLoginToken(t, r, *authController, models.LoginRequestModel{Email: ud.Email, Password: ud.Password})
		if token == "" {
			t.Fatalf("failed to get token for %s", tag)
		}
		return testUser{token: token, id: u.ID, orgID: orgID}
	}

	owner := newUser("owner", "", "")
	member := newUser("member", owner.orgID, "")
	recipient := newUser("recipient", owner.orgID, "")
	sameOrgAdminRole := createRoleWithPermission(t, db, owner.orgID, models.PermissionList{CanDeleteAnyFile: true})
	sameOrgAdmin := newUser("orgadmin", owner.orgID, sameOrgAdminRole)
	// Outsider holds can_delete_any_file through users.org_role_id, for their own organisation only.
	outsider := newUser("outsider", "", "")
	outsiderRole := createRoleWithPermission(t, db, outsider.orgID, models.PermissionList{CanDeleteAnyFile: true})
	db.Postgresql.Model(&models.User{}).Where("id = ?", outsider.id).Update("org_role_id", outsiderRole)
	// Member also carries an admin role id on users.org_role_id, which must no longer count.
	db.Postgresql.Model(&models.User{}).Where("id = ?", member.id).Update("org_role_id", outsiderRole)

	file := models.File{
		ID:             uuid.New().String(),
		FileName:       "edit_scope.txt",
		FileType:       "txt",
		MimeType:       "text/plain",
		FileLink:       "https://example.com/edit_scope.txt",
		UserID:         owner.id,
		OrganisationID: owner.orgID,
		AccessType:     "private",
	}
	if err := db.Postgresql.Create(&file).Error; err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	share := models.FileShare{
		ID:             uuid.New().String(),
		FileID:         file.ID,
		SharedByUserID: owner.id,
		OrganisationID: owner.orgID,
		AccessType:     "private",
		PermissionType: "edit",
		ShareLink:      "editscope-" + uuid.New().String(),
	}
	if err := db.Postgresql.Create(&share).Error; err != nil {
		t.Fatalf("failed to create share: %v", err)
	}
	if err := db.Postgresql.Create(&models.FileShareRecipient{FileShareID: share.ID, UserID: recipient.id}).Error; err != nil {
		t.Fatalf("failed to record recipient: %v", err)
	}
	defer func() {
		db.Postgresql.Where("file_share_id = ?", share.ID).Delete(&models.FileShareRecipient{})
		db.Postgresql.Unscoped().Where("id = ?", share.ID).Delete(&models.FileShare{})
		db.Postgresql.Unscoped().Where("id = ?", file.ID).Delete(&models.File{})
	}()

	rename := func(token, name string) int {
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/files/file/"+file.ID, strings.NewReader(`{"file_name":"`+name+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"member without a delivered share", member.token, http.StatusForbidden},
		{"admin of another organisation", outsider.token, http.StatusNotFound},
		{"recipient of the edit share", recipient.token, http.StatusOK},
		{"admin of the file's organisation", sameOrgAdmin.token, http.StatusOK},
		{"owner", owner.token, http.StatusOK},
	}
	for i, tc := range cases {
		if got := rename(tc.token, fmt.Sprintf("edit_scope_%d.txt", i)); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}

	// An expired share no longer grants edit rights.
	past := time.Now().UTC().Add(-time.Hour)
	db.Postgresql.Model(&models.FileShare{}).Where("id = ?", share.ID).Update("expires_at", past)
	if got := rename(recipient.token, "edit_scope_expired.txt"); got != http.StatusForbidden {
		t.Errorf("recipient of expired share: got %d, want %d", got, http.StatusForbidden)
	}
}
