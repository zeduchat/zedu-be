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

// TestUpdateFileShareAuthorization covers #1548: PUT /files/shares/:id may only be
// used by the user who created the share or the owner of the shared file.
func TestUpdateFileShareAuthorization(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	type testUser struct {
		token, id, orgID string
	}
	newUser := func(tag string) testUser {
		currUUID := uuid.New().String()
		ud := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("updshare_%s_%v@qa.team", tag, currUUID),
			FirstName:   "Share",
			LastName:    "Update" + tag,
			PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
			Password:    "password123",
			UserName:    fmt.Sprintf("updshare_%s_%v", tag, currUUID),
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
	owner, other := newUser("owner"), newUser("other")

	file := models.File{
		ID:             uuid.New().String(),
		FileName:       "upd_share.txt",
		FileType:       "txt",
		MimeType:       "text/plain",
		FileLink:       "https://example.com/upd_share.txt",
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
		PermissionType: "view",
		ShareLink:      "updshare-" + uuid.New().String(),
	}
	if err := db.Postgresql.Create(&share).Error; err != nil {
		t.Fatalf("failed to create share: %v", err)
	}
	defer func() {
		db.Postgresql.Unscoped().Where("id = ?", share.ID).Delete(&models.FileShare{})
		db.Postgresql.Unscoped().Where("id = ?", file.ID).Delete(&models.File{})
	}()

	put := func(token, body string) int {
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/files/shares/"+share.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr.Code
	}

	if got := put(other.token, `{"access_type":"public","permission_type":"edit"}`); got != http.StatusForbidden {
		t.Errorf("non-owner update: got %d, want %d", got, http.StatusForbidden)
	}
	var after models.FileShare
	db.Postgresql.Where("id = ?", share.ID).First(&after)
	if after.AccessType != "private" || after.PermissionType != "view" {
		t.Errorf("denied update changed the share: access=%s permission=%s", after.AccessType, after.PermissionType)
	}

	// A share created by someone else on the owner's file: the file owner may still change it.
	foreignShare := models.FileShare{
		ID:             uuid.New().String(),
		FileID:         file.ID,
		SharedByUserID: other.id,
		OrganisationID: owner.orgID,
		AccessType:     "private",
		PermissionType: "view",
		ShareLink:      "updshare-" + uuid.New().String(),
	}
	if err := db.Postgresql.Create(&foreignShare).Error; err != nil {
		t.Fatalf("failed to create share: %v", err)
	}
	defer db.Postgresql.Unscoped().Where("id = ?", foreignShare.ID).Delete(&models.FileShare{})
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/files/shares/"+foreignShare.ID, strings.NewReader(`{"note":"owner edit"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+owner.token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("file owner update of another user's share: got %d, want %d", rr.Code, http.StatusOK)
	}

	if got := put(owner.token, `{"access_type":"public","permission_type":"edit"}`); got != http.StatusOK {
		t.Errorf("share creator update: got %d, want %d", got, http.StatusOK)
	}
	db.Postgresql.Where("id = ?", share.ID).First(&after)
	if after.AccessType != "public" || after.PermissionType != "edit" {
		t.Errorf("creator update not applied: access=%s permission=%s", after.AccessType, after.PermissionType)
	}
}
