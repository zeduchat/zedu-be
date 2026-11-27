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

// TestGetFileDetailsAccess covers #1430: GET /files/file/:id must only return a file to
// its owner, to users in its organisation when it is public, and to members of the
// channel or DM it was posted in. Trashed files are owner-only. Everyone else gets 404.
func TestGetFileDetailsAccess(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	type testUser struct {
		token, id, orgID string
	}
	newUser := func(tag string) testUser {
		currUUID := uuid.New().String()
		ud := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("fileacc_%s_%v@qa.team", tag, currUUID),
			FirstName:   "File",
			LastName:    "Access" + tag,
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("fileacc_%s_%v", tag, currUUID),
		}
		tests.SignupUser(t, r, *authController, ud, false)
		token := tests.GetLoginToken(t, r, *authController, models.LoginRequestModel{Email: ud.Email, Password: ud.Password})
		if token == "" {
			t.Fatalf("failed to get token for %s", tag)
		}
		var u models.User
		db.Postgresql.Preload("Organisations").Where("email = ?", ud.Email).First(&u)
		if len(u.Organisations) == 0 {
			t.Fatalf("user %s has no organisation", tag)
		}
		return testUser{token: token, id: u.ID, orgID: u.Organisations[0].ID}
	}
	owner, other, outsider := newUser("owner"), newUser("other"), newUser("outsider")

	// DM between owner and other; outsider is not in it.
	dmID := uuid.New().String()
	if err := db.Postgresql.Create(&models.DmChannels{
		ID: uuid.New().String(), UserId: owner.id, ParticipantId: &other.id, ChannelId: dmID,
		OrgId: owner.orgID, ChannelType: "dm", ChatType: "user",
	}).Error; err != nil {
		t.Fatalf("failed to create dm: %v", err)
	}

	newFile := func(name, orgID, access string, channelID *string) models.File {
		f := models.File{
			ID:             uuid.New().String(),
			FileName:       name,
			FileType:       "txt",
			MimeType:       "text/plain",
			FileLink:       "https://example.com/" + name,
			UserID:         owner.id,
			OrganisationID: orgID,
			ChannelID:      channelID,
			AccessType:     access,
		}
		if err := db.Postgresql.Create(&f).Error; err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		return f
	}

	private := newFile("private.txt", owner.orgID, "private", nil)
	// Public file in the org of "other"'s token, and a public file only in owner's org.
	publicInOtherOrg := newFile("public_other_org.txt", other.orgID, "public", nil)
	publicInOwnerOrg := newFile("public_owner_org.txt", owner.orgID, "public", nil)
	inDM := newFile("in_dm.txt", owner.orgID, "private", &dmID)
	shared := newFile("shared.txt", owner.orgID, "private", nil)
	if err := db.Postgresql.Create(&models.FileShare{
		ID: uuid.New().String(), FileID: shared.ID, SharedByUserID: owner.id, OrganisationID: owner.orgID,
		AccessType: "private", PermissionType: "view", ShareLink: "fileacc-" + uuid.New().String(),
	}).Error; err != nil {
		t.Fatalf("failed to create share: %v", err)
	}
	trashed := newFile("trashed.txt", owner.orgID, "private", nil)
	if err := db.Postgresql.Delete(&trashed).Error; err != nil {
		t.Fatalf("failed to trash file: %v", err)
	}

	get := func(fileID, token string) int {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/file/"+fileID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}

	cases := []struct {
		name   string
		fileID string
		user   testUser
		want   int
	}{
		{"owner reads private file", private.ID, owner, http.StatusOK},
		{"other user reads private file", private.ID, other, http.StatusNotFound},
		{"same-org user reads public file", publicInOtherOrg.ID, other, http.StatusOK},
		{"other-org user reads public file", publicInOwnerOrg.ID, other, http.StatusNotFound},
		{"dm member reads file posted in dm", inDM.ID, other, http.StatusOK},
		{"non-member reads file posted in dm", inDM.ID, outsider, http.StatusNotFound},
		{"active share does not grant access", shared.ID, outsider, http.StatusNotFound},
		{"owner reads trashed file", trashed.ID, owner, http.StatusOK},
		{"other user reads trashed file", trashed.ID, other, http.StatusNotFound},
	}
	for _, tc := range cases {
		if got := get(tc.fileID, tc.user.token); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}
