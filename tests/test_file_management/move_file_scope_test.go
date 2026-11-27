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

// TestMoveFileScope covers #2510: PUT /files/:id/move only moves a file of the caller's
// organisation, and only into a folder the caller owns in that same organisation.
func TestMoveFileScope(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	type testUser struct {
		token, id, orgID string
	}
	newUser := func(tag, joinOrg string) testUser {
		currUUID := uuid.New().String()
		ud := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("movescope_%s_%v@qa.team", tag, currUUID),
			FirstName:   "Move",
			LastName:    "Scope" + tag,
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("movescope_%s_%v", tag, currUUID),
		}
		tests.SignupUser(t, r, *authController, ud, false)
		var u models.User
		db.Postgresql.Preload("Organisations").Where("email = ?", ud.Email).First(&u)
		if len(u.Organisations) == 0 {
			t.Fatalf("user %s has no organisation", tag)
		}
		orgID := u.Organisations[0].ID
		if joinOrg != "" {
			var role models.OrgRole
			if err := db.Postgresql.Where("name = ?", "User").First(&role).Error; err != nil {
				t.Fatalf("failed to find member role: %v", err)
			}
			if err := db.Postgresql.Create(&models.OrgUserManagement{
				UserID: u.ID, OrganisationID: joinOrg, RoleID: role.ID, Status: "active",
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
	owner := newUser("owner", "")
	outsider := newUser("outsider", "")
	member := newUser("member", owner.orgID)

	// The outsider is an administrator of their own organisation: users.org_role_id
	// carries a role with can_delete_any_file, which CheckFileEditPermission honours.
	adminRoleID := createRoleWithPermission(t, db, outsider.orgID, models.PermissionList{CanDeleteAnyFile: true})
	db.Postgresql.Model(&models.User{}).Where("id = ?", outsider.id).Update("org_role_id", adminRoleID)

	newFolder := func(u testUser) models.Folder {
		f := models.Folder{ID: uuid.New().String(), OrganisationID: u.orgID, UserID: u.id, Name: "movescope-" + uuid.New().String()[:8]}
		if err := db.Postgresql.Create(&f).Error; err != nil {
			t.Fatalf("failed to create folder: %v", err)
		}
		return f
	}
	ownerFolder, outsiderFolder, memberFolder := newFolder(owner), newFolder(outsider), newFolder(member)

	file := models.File{
		ID:             uuid.New().String(),
		FileName:       "move_scope.txt",
		FileType:       "txt",
		MimeType:       "text/plain",
		FileLink:       "https://example.com/move_scope.txt",
		UserID:         owner.id,
		OrganisationID: owner.orgID,
		AccessType:     "private",
	}
	if err := db.Postgresql.Create(&file).Error; err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer func() {
		db.Postgresql.Unscoped().Where("id = ?", file.ID).Delete(&models.File{})
		db.Postgresql.Unscoped().Where("id IN ?", []string{ownerFolder.ID, outsiderFolder.ID, memberFolder.ID}).Delete(&models.Folder{})
	}()

	move := func(token, folderID string) int {
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/files/"+file.ID+"/move", strings.NewReader(`{"folder_id":"`+folderID+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}
	folderOf := func() string {
		var f models.File
		db.Postgresql.Where("id = ?", file.ID).First(&f)
		if f.FolderID == nil {
			return ""
		}
		return *f.FolderID
	}

	cases := []struct {
		name     string
		token    string
		folderID string
		want     int
	}{
		{"owner moves into another organisation's folder", owner.token, outsiderFolder.ID, http.StatusNotFound},
		{"owner moves into another member's folder", owner.token, memberFolder.ID, http.StatusForbidden},
		{"owner moves into missing folder", owner.token, uuid.New().String(), http.StatusNotFound},
		{"admin of another organisation moves the file", outsider.token, outsiderFolder.ID, http.StatusNotFound},
		{"owner moves into own folder", owner.token, ownerFolder.ID, http.StatusOK},
	}
	for _, tc := range cases {
		before := folderOf()
		got := move(tc.token, tc.folderID)
		if got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
		if tc.want != http.StatusOK && folderOf() != before {
			t.Errorf("%s: denied move changed folder_id", tc.name)
		}
	}
	if folderOf() != ownerFolder.ID {
		t.Errorf("file not in owner's folder after allowed move")
	}
}
