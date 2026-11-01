package test_file_management

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

type fileAccessUser struct {
	ID    string
	OrgID string
	Token string
}

func signUpFileAccessUser(t *testing.T, name string) fileAccessUser {
	t.Helper()
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%s%v@qa.team", name, currUUID),
		FirstName:   "Test",
		LastName:    "FileAccess",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("%s_%v", name, currUUID),
	}

	tests.SignupUser(t, r, *authController, signUp, false)
	token := tests.GetLoginToken(t, r, *authController, models.LoginRequestModel{Email: signUp.Email, Password: signUp.Password})
	if token == "" {
		t.Fatal("Failed to get authentication token")
	}

	var user models.User
	db.Postgresql.Preload("Organisations").Where("email = ?", signUp.Email).First(&user)
	if len(user.Organisations) == 0 {
		t.Fatal("User has no organisations")
	}

	return fileAccessUser{ID: user.ID, OrgID: user.Organisations[0].ID, Token: token}
}

func createAccessTestFile(t *testing.T, db *gorm.DB, ownerID, orgID, accessType string) models.File {
	t.Helper()
	file := models.File{
		ID:             uuid.New().String(),
		FileName:       "access_test.txt",
		FileType:       "txt",
		MimeType:       "text/plain",
		FileLink:       "https://example.com/access_test.txt",
		UserID:         ownerID,
		OrganisationID: orgID,
		AccessType:     accessType,
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return file
}

func getFileByID(t *testing.T, fileID, token string) *httptest.ResponseRecorder {
	t.Helper()
	r, _, _, _ := SetupFileManagementTestRouter()

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/file/%s", fileID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestGetFileDetailsByID_AccessControl(t *testing.T) {
	_, _, _, db := SetupFileManagementTestRouter()

	owner := signUpFileAccessUser(t, "fileowner")
	outsider := signUpFileAccessUser(t, "fileoutsider")

	t.Run("owner can read their own file", func(t *testing.T) {
		file := createAccessTestFile(t, db.Postgresql, owner.ID, owner.OrgID, "private")

		rr := getFileByID(t, file.ID, owner.Token)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("a user from another organisation gets a 404", func(t *testing.T) {
		file := createAccessTestFile(t, db.Postgresql, owner.ID, owner.OrgID, "private")

		rr := getFileByID(t, file.ID, outsider.Token)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d. Response: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("a public file is still not readable from another organisation", func(t *testing.T) {
		file := createAccessTestFile(t, db.Postgresql, owner.ID, owner.OrgID, "public")

		rr := getFileByID(t, file.ID, outsider.Token)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d. Response: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("a file with an active share is not readable from another organisation", func(t *testing.T) {
		file := createAccessTestFile(t, db.Postgresql, owner.ID, owner.OrgID, "private")
		share := models.FileShare{
			ID:             uuid.New().String(),
			FileID:         file.ID,
			SharedByUserID: owner.ID,
			OrganisationID: owner.OrgID,
			AccessType:     "private",
			PermissionType: "view",
			ShareLink:      uuid.New().String(),
		}
		if err := db.Postgresql.Create(&share).Error; err != nil {
			t.Fatalf("Failed to create share: %v", err)
		}

		rr := getFileByID(t, file.ID, outsider.Token)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d. Response: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("a private file of a colleague is forbidden, a public one is readable", func(t *testing.T) {
		colleagueID := uuid.New().String()
		private := createAccessTestFile(t, db.Postgresql, colleagueID, owner.OrgID, "private")
		public := createAccessTestFile(t, db.Postgresql, colleagueID, owner.OrgID, "public")

		if rr := getFileByID(t, private.ID, owner.Token); rr.Code != http.StatusForbidden {
			t.Fatalf("private file: expected 403, got %d. Response: %s", rr.Code, rr.Body.String())
		}
		if rr := getFileByID(t, public.ID, owner.Token); rr.Code != http.StatusOK {
			t.Fatalf("public file: expected 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("a deleted file is not returned, even to its owner", func(t *testing.T) {
		file := createAccessTestFile(t, db.Postgresql, owner.ID, owner.OrgID, "private")
		if err := db.Postgresql.Delete(&file).Error; err != nil {
			t.Fatalf("Failed to soft delete file: %v", err)
		}

		rr := getFileByID(t, file.ID, owner.Token)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d. Response: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("an invalid id is rejected", func(t *testing.T) {
		rr := getFileByID(t, "not-a-uuid", owner.Token)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}
	})
}
