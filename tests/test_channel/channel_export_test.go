package test_channel

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/minio/minio-go/v7"
	"github.com/riverqueue/river"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/middleware"
	riverqueueBg "github.com/hngprojects/telex_be/pkg/repository/riverqueue"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestChannelExportEndpoints(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create User & Auth Token
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser_export_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Export",
		LastName:    "Tester",
		Password:    "password",
		UserName:    fmt.Sprintf("exportuser_%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authController, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authController, loginData)

	var user models.User
	if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	// Create Test Channel
	channelID := utility.GenerateUUID()
	testChannel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("test-export-channel-%v", currUUID),
		Description:    "Channel for export testing",
		OwnerId:        user.ID,
		OrganisationID: org.ID,
	}
	if err := db.Postgresql.Create(&testChannel).Error; err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	// Add User to Channel
	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user.ID,
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	// Setup Channel Controller & Routes
	channelController := channel.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r.POST("/api/v1/channels/:channelId/export", middleware.Authorize(db.Postgresql), channelController.ExportChannel)
	r.GET("/api/v1/channels/:channelId/export/status", middleware.Authorize(db.Postgresql), channelController.GetChannelExportStatus)
	r.GET("/api/v1/channels/:channelId/export/history", middleware.Authorize(db.Postgresql), channelController.GetChannelExportHistory)

	var firstExportID string

	// Test 1: Trigger New Export
	t.Run("Initiate Channel Export", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%s/export", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Fatalf("Expected status 202, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data object in response: %v", response)
		}

		if data["status"] != string(models.ExportStatusPending) {
			t.Errorf("Expected export status 'pending', got %v", data["status"])
		}

		firstExportID, _ = data["id"].(string)
		if firstExportID == "" {
			t.Fatal("Export ID is empty")
		}
	})

	// Test 2: Trigger Duplicate Export while in-progress
	t.Run("Deduplicate Active Channel Export", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%s/export", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200 for duplicate request, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data object in response: %v", response)
		}

		if data["id"] != firstExportID {
			t.Errorf("Expected active export ID %s, got %v", firstExportID, data["id"])
		}
	})

	// Test 3: Get Export Status
	t.Run("Get Latest Channel Export Status", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/export/status", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data object in status response: %v", response)
		}

		if data["id"] != firstExportID {
			t.Errorf("Expected status export ID %s, got %v", firstExportID, data["id"])
		}
	})

	// Test 4: Get Export History
	t.Run("Get Channel Export History", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/export/history?page=1&limit=10", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data array in history response: %v", response)
		}

		if len(data) == 0 {
			t.Fatal("Expected export history records in response")
		}

		pagination, ok := response["pagination"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected pagination metadata in history response")
		}

		if pagination["current_page"] == nil {
			t.Error("Missing current_page in pagination metadata")
		}
	})
}

func TestChannelExportWorkerExecution(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	userID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()
	channelID := utility.GenerateUUID()

	testUser := models.User{
		ID:         userID,
		Email:      fmt.Sprintf("worker_test_%v@qa.team", currUUID),
		Password:   "password",
		IsVerified: true,
	}
	_ = db.Postgresql.Create(&testUser)

	testOrg := models.Organisation{
		ID:      orgID,
		Name:    fmt.Sprintf("Worker Org %v", currUUID),
		OwnerID: userID,
	}
	_ = db.Postgresql.Create(&testOrg)

	testChannel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("worker-channel-%v", currUUID),
		Description:    "Worker export test channel",
		OwnerId:        userID,
		OrganisationID: orgID,
	}
	_ = db.Postgresql.Create(&testChannel)

	exportRecord := models.ChannelExport{
		ID:             utility.GenerateUUID(),
		ChannelID:      channelID,
		UserID:         userID,
		OrganisationID: orgID,
		Status:         models.ExportStatusPending,
	}
	if err := db.Postgresql.Create(&exportRecord).Error; err != nil {
		t.Fatalf("Failed to create pending export record: %v", err)
	}

	worker := riverqueueBg.NewChannelExportWorker(logger, db.Postgresql)
	job := &river.Job[models.ChannelExportJobArgs]{
		Args: models.ChannelExportJobArgs{
			ExportID:       exportRecord.ID,
			ChannelID:      channelID,
			UserID:         userID,
			OrganisationID: orgID,
		},
	}

	_ = worker.Work(context.Background(), job)

	var updatedExport models.ChannelExport
	if err := db.Postgresql.Where("id = ?", exportRecord.ID).First(&updatedExport).Error; err != nil {
		t.Fatalf("Failed to fetch updated export record: %v", err)
	}

	if updatedExport.Status == models.ExportStatusPending {
		t.Errorf("Export status was not updated from pending")
	}
}

func TestChannelExportFullDataFlow(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	userID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()
	channelID := utility.GenerateUUID()
	threadID := utility.GenerateUUID()
	replyID := utility.GenerateUUID()
	mediaID := utility.GenerateUUID()

	testUser := models.User{
		ID:         userID,
		Email:      fmt.Sprintf("export_full_%v@qa.team", currUUID),
		Password:   "password",
		IsVerified: true,
	}
	_ = db.Postgresql.Create(&testUser)

	testOrg := models.Organisation{
		ID:      orgID,
		Name:    fmt.Sprintf("Export Full Org %v", currUUID),
		OwnerID: userID,
	}
	_ = db.Postgresql.Create(&testOrg)

	testChannel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("full-export-channel-%v", currUUID),
		Description:    "Channel with thread, reply, and media for full export testing",
		OwnerId:        userID,
		OrganisationID: orgID,
	}
	_ = db.Postgresql.Create(&testChannel)

	testFile := models.File{
		ID:             mediaID,
		FileName:       "sample_image.png",
		FileType:       "image",
		MimeType:       "image/png",
		FileLink:       "https://minio.example.com/bucket/public/file-uploads/sample_image.png",
		Size:           1024,
		OrganisationID: orgID,
		UserID:         userID,
		ChannelID:      &channelID,
	}
	_ = db.Postgresql.Create(&testFile)

	if db.Elastic != nil {
		threadDoc := map[string]interface{}{
			"thread_id":     threadID,
			"channels_id":   channelID,
			"org_id":        orgID,
			"user_id":       userID,
			"username":      "rootuser",
			"full_name":     "Root User",
			"message":       "Root thread content for export testing",
			"type":          "message",
			"message_count": 1,
			"created_at":    time.Now().Format(time.RFC3339),
			"media": []interface{}{
				map[string]interface{}{
					"id":        mediaID,
					"file_name": "sample_image.png",
					"file_type": "image",
					"file_link": testFile.FileLink,
				},
			},
		}
		_ = elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, threadDoc, logger)

		replyDoc := map[string]interface{}{
			"id":          replyID,
			"thread_id":   threadID,
			"channels_id": channelID,
			"org_id":      orgID,
			"user_id":     userID,
			"username":    "replyuser",
			"full_name":   "Reply User",
			"message":     "Complete reply message for export testing",
			"created_at":  time.Now().Format(time.RFC3339),
		}
		_ = elastic.AddDocument(db.Elastic, models.MessageIndexName, replyID, replyDoc, logger)

		time.Sleep(1 * time.Second)
	}

	exportRecord := models.ChannelExport{
		ID:             utility.GenerateUUID(),
		ChannelID:      channelID,
		UserID:         userID,
		OrganisationID: orgID,
		Status:         models.ExportStatusPending,
	}
	if err := db.Postgresql.Create(&exportRecord).Error; err != nil {
		t.Fatalf("Failed to create pending export record: %v", err)
	}

	worker := riverqueueBg.NewChannelExportWorker(logger, db.Postgresql)
	job := &river.Job[models.ChannelExportJobArgs]{
		Args: models.ChannelExportJobArgs{
			ExportID:       exportRecord.ID,
			ChannelID:      channelID,
			UserID:         userID,
			OrganisationID: orgID,
		},
	}

	_ = worker.Work(context.Background(), job)

	var updatedExport models.ChannelExport
	if err := db.Postgresql.Where("id = ?", exportRecord.ID).First(&updatedExport).Error; err != nil {
		t.Fatalf("Failed to fetch updated export record: %v", err)
	}

	t.Logf("Worker execution status: %s, ErrorMessage: %v", updatedExport.Status, updatedExport.ErrorMessage)

	if updatedExport.Status == models.ExportStatusCompleted {
		if updatedExport.FileURL == nil || *updatedExport.FileURL == "" {
			t.Error("Completed export has empty FileURL")
		}
		if updatedExport.FileID == nil || *updatedExport.FileID == "" {
			t.Error("Completed export has empty FileID")
		}

		var createdFile models.File
		if err := db.Postgresql.Where("id = ?", *updatedExport.FileID).First(&createdFile).Error; err != nil {
			t.Errorf("File record was not created in files table: %v", err)
		} else {
			if createdFile.FileType != "zip" {
				t.Errorf("Expected file_type 'zip', got %s", createdFile.FileType)
			}
		}
	}
}

func TestChannelExportZipContentsLocal(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	userID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()
	channelID := utility.GenerateUUID()
	threadID := utility.GenerateUUID()
	replyID := utility.GenerateUUID()
	mediaID := utility.GenerateUUID()

	testUser := models.User{
		ID:         userID,
		Email:      fmt.Sprintf("export_zip_content_%v@qa.team", currUUID),
		Password:   "password",
		IsVerified: true,
	}
	_ = db.Postgresql.Create(&testUser)

	testOrg := models.Organisation{
		ID:      orgID,
		Name:    fmt.Sprintf("Export Zip Content Org %v", currUUID),
		OwnerID: userID,
	}
	_ = db.Postgresql.Create(&testOrg)

	testChannel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("zip-content-channel-%v", currUUID),
		Description:    "Channel for verifying unzipped contents",
		OwnerId:        userID,
		OrganisationID: orgID,
	}
	_ = db.Postgresql.Create(&testChannel)

	testFile := models.File{
		ID:             mediaID,
		FileName:       "test_attachment.png",
		FileType:       "image",
		MimeType:       "image/png",
		FileLink:       "https://minio.example.com/bucket/public/file-uploads/test_attachment.png",
		Size:           512,
		OrganisationID: orgID,
		UserID:         userID,
		ChannelID:      &channelID,
	}
	_ = db.Postgresql.Create(&testFile)

	if db.Elastic != nil {
		threadDoc := map[string]interface{}{
			"thread_id":     threadID,
			"channels_id":   channelID,
			"org_id":        orgID,
			"user_id":       userID,
			"username":      "threadowner",
			"full_name":     "Thread Owner",
			"message":       "Root thread content for zip inspection",
			"type":          "message",
			"message_count": 1,
			"created_at":    time.Now().Format(time.RFC3339),
			"media": []interface{}{
				map[string]interface{}{
					"id":        mediaID,
					"file_name": "test_attachment.png",
					"file_type": "image",
					"file_link": testFile.FileLink,
				},
			},
		}
		_ = elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, threadDoc, logger)

		replyDoc := map[string]interface{}{
			"id":          replyID,
			"thread_id":   threadID,
			"channels_id": channelID,
			"org_id":      orgID,
			"user_id":     userID,
			"username":    "replier",
			"full_name":   "Replier User",
			"message":     "Reply content for zip inspection",
			"created_at":  time.Now().Format(time.RFC3339),
		}
		_ = elastic.AddDocument(db.Elastic, models.MessageIndexName, replyID, replyDoc, logger)

		time.Sleep(1 * time.Second)
	}

	exportRecord := models.ChannelExport{
		ID:             utility.GenerateUUID(),
		ChannelID:      channelID,
		UserID:         userID,
		OrganisationID: orgID,
		Status:         models.ExportStatusPending,
	}
	if err := db.Postgresql.Create(&exportRecord).Error; err != nil {
		t.Fatalf("Failed to create pending export record: %v", err)
	}

	worker := riverqueueBg.NewChannelExportWorker(logger, db.Postgresql)
	job := &river.Job[models.ChannelExportJobArgs]{
		Args: models.ChannelExportJobArgs{
			ExportID:       exportRecord.ID,
			ChannelID:      channelID,
			UserID:         userID,
			OrganisationID: orgID,
		},
	}

	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("Worker execution failed: %v", err)
	}

	var updatedExport models.ChannelExport
	if err := db.Postgresql.Where("id = ?", exportRecord.ID).First(&updatedExport).Error; err != nil {
		t.Fatalf("Failed to fetch updated export record: %v", err)
	}

	if updatedExport.Status != models.ExportStatusCompleted {
		t.Fatalf("Expected export status completed, got %s (error: %v)", updatedExport.Status, updatedExport.ErrorMessage)
	}

	if updatedExport.FileURL == nil || *updatedExport.FileURL == "" {
		t.Fatal("Export FileURL is empty")
	}

	t.Logf("Export generated successfully with FileURL: %s", *updatedExport.FileURL)

	zr, cleanup := verifyAndUnpackExportZIP(t, *updatedExport.FileURL)
	defer cleanup()

	if zr != nil {
		verifyExportZipEntries(t, zr)
	}
}

func verifyAndUnpackExportZIP(t *testing.T, fileURL string) (*zip.Reader, func()) {
	if storage.DB == nil || storage.DB.Minio == nil {
		t.Log("MinIO client not initialized, skipping ZIP byte verification")
		return nil, func() {}
	}

	zipFileName := utility.ExtractHashedFileName(fileURL)
	if zipFileName == "" {
		t.Error("Failed to extract hashed filename from export FileURL")
		return nil, func() {}
	}

	objectKey := "public/file-uploads/exports/" + zipFileName
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	cleanupFunc := func() {
		cleanupExportZipFromMinIO(t, objectKey)
	}

	obj, err := minioClient.GetObject(context.Background(), bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		t.Errorf("Failed to get export ZIP object from MinIO: %v", err)
		return nil, cleanupFunc
	}
	defer obj.Close()

	zipBuf, err := io.ReadAll(obj)
	if err != nil || len(zipBuf) == 0 {
		t.Errorf("Failed to read export ZIP bytes: %v", err)
		return nil, cleanupFunc
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBuf), int64(len(zipBuf)))
	if err != nil {
		t.Fatalf("Failed to parse generated ZIP archive: %v", err)
		return nil, cleanupFunc
	}

	return zr, cleanupFunc
}

func cleanupExportZipFromMinIO(t *testing.T, objectKey string) {
	if storage.DB == nil || storage.DB.Minio == nil {
		return
	}
	bucketName := config.Config.Minio.BucketName
	err := storage.DB.Minio.RemoveObject(context.Background(), bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		t.Logf("Warning: Failed to cleanup export ZIP object %s from MinIO: %v", objectKey, err)
	} else {
		t.Logf("Successfully cleaned up export ZIP object from MinIO: %s", objectKey)
	}
}

func verifyChannelInfoMetadata(t *testing.T, content []byte) {
	var meta map[string]interface{}
	if err := json.Unmarshal(content, &meta); err != nil {
		t.Errorf("channel_info.json is invalid JSON: %v", err)
		return
	}
	t.Logf("channel_info.json contents: %+v", meta)
	if totalThreads, _ := meta["total_threads"].(float64); totalThreads < 1 {
		t.Errorf("Expected total_threads >= 1 in channel_info.json, got %v", meta["total_threads"])
	}
}

func verifyThreadsMetadata(t *testing.T, content []byte) {
	var threads []models.ThreadDocument
	if err := json.Unmarshal(content, &threads); err != nil {
		t.Errorf("threads.json is invalid JSON: %v", err)
		return
	}
	t.Logf("threads.json extracted %d threads", len(threads))
	if len(threads) < 1 {
		t.Errorf("Expected threads.json to contain >= 1 threads, got 0")
	}
}

func verifyExportZipEntries(t *testing.T, zr *zip.Reader) {
	var foundMeta, foundThreads bool
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, _ := io.ReadAll(rc)
		_ = rc.Close()

		if f.Name == "channel_info.json" {
			foundMeta = true
			verifyChannelInfoMetadata(t, content)
		}
		if f.Name == "threads.json" {
			foundThreads = true
			verifyThreadsMetadata(t, content)
		}
	}

	if !foundMeta {
		t.Error("channel_info.json missing from exported ZIP")
	}
	if !foundThreads {
		t.Error("threads.json missing from exported ZIP")
	}
}
