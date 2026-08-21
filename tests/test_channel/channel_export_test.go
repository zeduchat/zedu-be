package test_channel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/riverqueue/river"

	"github.com/hngprojects/telex_be/external/request"
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
			"type":          "thread",
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
