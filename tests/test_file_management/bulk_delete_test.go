package test_file_management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
	"github.com/stretchr/testify/assert"
)

func TestBulkDelete(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	user := models.CreateUserRequestModel{
		Email:       "test@qa.team",
		Password:    "password",
		FirstName:   "Test",
		LastName:    "User",
		UserName:    "testuser",
		PhoneNumber: "1234567890",
	}
	tests.SignupUser(t, r, *authController, user, false)

	loginData := models.LoginRequestModel{
		Email:    "test@qa.team",
		Password: "password",
	}
	token := tests.GetLoginToken(t, r, *authController, loginData)

	t.Run("BulkDeleteFiles_Soft", func(t *testing.T) {

		fileID1 := uploadTestFile(t, r, token, "file1.txt", "text/plain", []byte("content1"))
		fileID2 := uploadTestFile(t, r, token, "file2.txt", "text/plain", []byte("content2"))

		reqBody := models.DeleteMultipleFilesRequest{
			IDs:       []string{fileID1, fileID2},
			Permanent: false,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/files", bytes.NewBuffer(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify files are soft deleted
		var file1, file2 models.File
		err1 := db.Postgresql.Unscoped().Where("id = ?", fileID1).First(&file1).Error
		err2 := db.Postgresql.Unscoped().Where("id = ?", fileID2).First(&file2).Error

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotNil(t, file1.DeletedAt)
		assert.NotNil(t, file2.DeletedAt)
	})

	t.Run("BulkDeleteFiles_Permanent", func(t *testing.T) {

		fileID1 := uploadTestFile(t, r, token, "file3.txt", "text/plain", []byte("content3"))
		fileID2 := uploadTestFile(t, r, token, "file4.txt", "text/plain", []byte("content4"))

		reqBody := models.DeleteMultipleFilesRequest{
			IDs:       []string{fileID1, fileID2},
			Permanent: true,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/files", bytes.NewBuffer(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify files are permanently deleted
		var file1, file2 models.File
		err1 := db.Postgresql.Unscoped().Where("id = ?", fileID1).First(&file1).Error
		err2 := db.Postgresql.Unscoped().Where("id = ?", fileID2).First(&file2).Error

		assert.Error(t, err1)
		assert.Error(t, err2)
	})

	t.Run("BulkDeleteFolders_Soft", func(t *testing.T) {

		folderID1 := createTestFolder(t, r, token, "Folder1")
		folderID2 := createTestFolder(t, r, token, "Folder2")

		reqBody := models.DeleteMultipleFoldersRequest{
			IDs:       []string{folderID1, folderID2},
			Permanent: false,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/files/folders", bytes.NewBuffer(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify folders are soft deleted
		var folder1, folder2 models.Folder
		err1 := db.Postgresql.Unscoped().Where("id = ?", folderID1).First(&folder1).Error
		err2 := db.Postgresql.Unscoped().Where("id = ?", folderID2).First(&folder2).Error

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotNil(t, folder1.DeletedAt)
		assert.NotNil(t, folder2.DeletedAt)
	})

	t.Run("BulkDeleteFolders_Permanent", func(t *testing.T) {

		folderID1 := createTestFolder(t, r, token, "Folder3")
		folderID2 := createTestFolder(t, r, token, "Folder4")

		reqBody := models.DeleteMultipleFoldersRequest{
			IDs:       []string{folderID1, folderID2},
			Permanent: true,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/files/folders", bytes.NewBuffer(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tests.AssertStatusCode(t, rr.Code, http.StatusOK)

		// verify folders are permanently deleted
		var folder1, folder2 models.Folder
		err1 := db.Postgresql.Unscoped().Where("id = ?", folderID1).First(&folder1).Error
		err2 := db.Postgresql.Unscoped().Where("id = ?", folderID2).First(&folder2).Error

		assert.Error(t, err1)
		assert.Error(t, err2)
	})
}

func createTestFolder(t *testing.T, r *gin.Engine, token string, name string) string {
	folder := models.CreateFolderParams{
		Name: name,
	}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(folder)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/folders", &b)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tests.AssertStatusCode(t, rr.Code, http.StatusCreated)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if _, ok := resp["data"]; !ok {
		t.Fatalf("Failed to create folder. Status: %d, Response: %s", rr.Code, rr.Body.String())
	}

	data := resp["data"].(map[string]interface{})
	return data["id"].(string)
}
