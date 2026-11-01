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
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
)

// Ziki #1215: a user must not be able to delete another user's folder.
func TestFolderDeleteOwnership(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	ownerToken := signupAndLogin(t, r, authController, "owner")
	otherToken := signupAndLogin(t, r, authController, "other")

	folderStillThere := func(t *testing.T, folderID string) {
		var folder models.Folder
		err := db.Postgresql.Where("id = ?", folderID).First(&folder).Error
		assert.NoError(t, err, "the folder should not have been deleted")
	}

	t.Run("OtherUserCannotDeleteFolder", func(t *testing.T) {
		folderID := createTestFolder(t, r, ownerToken, "Owner folder 1")

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s?permanent=true", folderID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", otherToken))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, rr.Code)
		folderStillThere(t, folderID)
	})

	t.Run("OtherUserCannotBulkDeleteFolders", func(t *testing.T) {
		folderID := createTestFolder(t, r, ownerToken, "Owner folder 2")

		body, _ := json.Marshal(models.DeleteMultipleFoldersRequest{IDs: []string{folderID}, Permanent: true})
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/files/folders", bytes.NewBuffer(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", otherToken))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.NotEqual(t, http.StatusOK, rr.Code)
		folderStillThere(t, folderID)
	})

	t.Run("OwnerCanDeleteFolder", func(t *testing.T) {
		folderID := createTestFolder(t, r, ownerToken, "Owner folder 3")

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s?permanent=true", folderID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ownerToken))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		var folder models.Folder
		err := db.Postgresql.Unscoped().Where("id = ?", folderID).First(&folder).Error
		assert.Error(t, err)
	})

	t.Run("OwnerCanPermanentlyDeleteFolderInTrash", func(t *testing.T) {
		folderID := createTestFolder(t, r, ownerToken, "Owner folder 4")

		for _, permanent := range []bool{false, true} {
			req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s?permanent=%t", folderID, permanent), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ownerToken))
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			tests.AssertStatusCode(t, rr.Code, http.StatusOK)
		}

		var folder models.Folder
		err := db.Postgresql.Unscoped().Where("id = ?", folderID).First(&folder).Error
		assert.Error(t, err)
	})

	t.Run("UnknownFolderIsNotFound", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/files/folders/%s", uuid.New().String()), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ownerToken))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})
}

func signupAndLogin(t *testing.T, r *gin.Engine, authController *auth.Controller, prefix string) string {
	currUUID := uuid.New().String()
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("test_%s_%s@qa.team", prefix, currUUID),
		Password:    "password",
		FirstName:   "Test",
		LastName:    "Owner",
		UserName:    fmt.Sprintf("test%s%s", prefix, currUUID),
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	tests.SignupUser(t, r, *authController, user, false)

	return tests.GetLoginToken(t, r, *authController, models.LoginRequestModel{Email: user.Email, Password: "password"})
}
