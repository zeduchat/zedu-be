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

// TestGetFileInfoAccess covers #1660: GET /files/file/:id/info must only return file info
// (owner, channels) to callers who may read the file. Everyone else gets 404.
func TestGetFileInfoAccess(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	type testUser struct {
		token, id, orgID string
	}
	newUser := func(tag string) testUser {
		currUUID := uuid.New().String()
		ud := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("fileinfoacc_%s_%v@qa.team", tag, currUUID),
			FirstName:   "FileInfo",
			LastName:    "Access" + tag,
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("fileinfoacc_%s_%v", tag, currUUID),
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

	private := newFile("info_private.txt", owner.orgID, "private", nil)
	publicInOtherOrg := newFile("info_public_other_org.txt", other.orgID, "public", nil)
	publicInOwnerOrg := newFile("info_public_owner_org.txt", owner.orgID, "public", nil)
	inDM := newFile("info_in_dm.txt", owner.orgID, "private", &dmID)

	get := func(fileID, token string) int {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/file/"+fileID+"/info", nil)
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
		{"owner reads private file info", private.ID, owner, http.StatusOK},
		{"other user reads private file info", private.ID, other, http.StatusNotFound},
		{"same-org user reads public file info", publicInOtherOrg.ID, other, http.StatusOK},
		{"other-org user reads public file info", publicInOwnerOrg.ID, other, http.StatusNotFound},
		{"dm member reads info of file posted in dm", inDM.ID, other, http.StatusOK},
		{"non-member reads info of file posted in dm", inDM.ID, outsider, http.StatusNotFound},
	}
	for _, tc := range cases {
		if got := get(tc.fileID, tc.user.token); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}
