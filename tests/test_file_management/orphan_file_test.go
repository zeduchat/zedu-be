package test_file_management

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
)

func TestOrphanFileRecovery(t *testing.T) {
	r, _, authController, db := SetupFileManagementTestRouter()

	currUUID := uuid.New().String()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testorphan%v@qa.team", currUUID),
		FirstName:   "Test",
		LastName:    "OrphanUser",
		PhoneNumber: fmt.Sprintf("%d", time.Now().UnixNano()),
		Password:    "password123",
		UserName:    fmt.Sprintf("test_orphan%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	tests.SignupUser(t, r, *authController, userSignUpData, false)
	token := tests.GetLoginToken(t, r, *authController, loginData)

	if token == "" {
		t.Fatal("Failed to get authentication token")
	}

	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName
	fileName := "orphan_test_file.txt"
	fileContent := []byte("This is an orphan file content")

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", fileName)
	part.Write(fileContent)
	writer.Close()

	var fileID string
	var fileLink string

	t.Run("Setup_CreateOrphanFile", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", fileName)
		if err != nil {
			t.Fatal(err)
		}
		part.Write(fileContent)
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-files", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Setup upload failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		files := resp["data"].([]interface{})
		f := files[0].(map[string]interface{})
		fileID = f["id"].(string)
		fileLink = f["file_link"].(string)

		//  delete the DB record but keep the MinIO file
		err = db.Postgresql.Unscoped().Where("id = ?", fileID).Delete(&models.File{}).Error
		if err != nil {
			t.Fatalf("Failed to delete DB record: %v", err)
		}

		// verify file still exists in MinIO
		hashedFileName := filepath.Base(fileLink)
		encodedFilePath := "public/file-uploads/" + hashedFileName
		_, err = minioClient.StatObject(context.Background(), bucketName, encodedFilePath, minio.StatObjectOptions{})
		if err != nil {
			t.Fatalf("File should exist in MinIO after DB deletion: %v", err)
		}
	})

	t.Run("RecoverOrphanFile", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", fileName)
		if err != nil {
			t.Fatal(err)
		}
		part.Write(fileContent)
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload-files", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Recovery upload failed with status %d. Response: %s", rr.Code, rr.Body.String())
		}

		resp := tests.ParseResponse(rr)
		files := resp["data"].([]interface{})
		f := files[0].(map[string]interface{})
		newFileID := f["id"].(string)

		if newFileID == fileID {
			t.Log("Warning: New file ID matches old deleted ID")
		}

		// verify DB record exists
		var file models.File
		err = db.Postgresql.Where("id = ?", newFileID).First(&file).Error
		if err != nil {
			t.Fatalf("New DB record not found: %v", err)
		}

		if file.FileLink != fileLink {
			t.Errorf("Expected file link to match original: %s, got %s", fileLink, file.FileLink)
		}
	})
}
