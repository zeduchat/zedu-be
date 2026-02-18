package test_message

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	dm "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetDmChannelMediaExtended(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create User 1
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser1_dm_media_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Test",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("testuser1_dm_%v", currUUID),
	}
	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authController, user1SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

	var user1 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	// Create User 2
	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser2_dm_media_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Test",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("testuser2_dm_%v", currUUID),
	}

	tst.SignupUser(t, gin.Default(), authController, user2SignUpData, false)
	var user2 models.User
	if err := db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error; err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}

	// Setup DM Controller
	dmController := dm.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r.GET("/api/v1/organisations/:org_id/dms/:channel_id/media", middleware.Authorize(db.Postgresql), dmController.GetDmChannelMedia)

	t.Run("Get Single DM Channel Media", func(t *testing.T) {
		// Create Single DM Channel
		dmChannelID := utility.GenerateUUID()
		participantID := user2.ID
		dmChannel := models.DmChannels{
			ID:            utility.GenerateUUID(),
			UserId:        user1.ID,
			ChannelId:     dmChannelID,
			OrgId:         org.ID,
			ParticipantId: &participantID,
			ChatType:      "user",
			ChannelType:   "dm",
		}

		if err := db.Postgresql.Create(&dmChannel).Error; err != nil {
			t.Fatalf("Failed to create DM channel: %v", err)
		}

		// Create Thread with Media in Elasticsearch
		threadID := utility.GenerateUUID()
		mediaID := utility.GenerateUUID()
		mediaFiles := []map[string]any{
			{
				"id":        mediaID,
				"file_name": "test_dm_file.pdf",
				"file_type": "pdf",
				"mime_type": "application/pdf",
				"file_link": "http://example.com/test_dm_file.pdf",
			},
		}

		thread := map[string]any{
			"id":          threadID,
			"thread_id":   threadID,
			"channels_id": dmChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Test DM message with file",
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
			"media":       mediaFiles,
		}

		// Index and wait
		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}
		time.Sleep(2 * time.Second)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/media", org.ID, dmChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].([]interface{})
		if !ok || len(data) == 0 {
			t.Fatal("Expected files in response data")
		}

		found := false
		for _, item := range data {
			file := item.(map[string]interface{})
			if file["id"] == mediaID {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected file ID %s not found in response", mediaID)
		}
	})

	t.Run("Get Group DM Channel Media", func(t *testing.T) {
		// Create Group DM Channel
		groupDmChannelID := utility.GenerateUUID()
		groupDmChannel := models.DmChannels{
			ID:          utility.GenerateUUID(),
			UserId:      user1.ID,
			ChannelId:   groupDmChannelID,
			OrgId:       org.ID,
			ChatType:    "user",
			ChannelType: "group_dm",
		}

		if err := db.Postgresql.Create(&groupDmChannel).Error; err != nil {
			t.Fatalf("Failed to create Group DM channel: %v", err)
		}

		// Add User1 and User2 as participants
		user1Participant := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDmChannelID,
			UserId:    user1.ID,
			OrgId:     org.ID,
		}
		user2Participant := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDmChannelID,
			UserId:    user2.ID,
			OrgId:     org.ID,
		}

		if err := db.Postgresql.Create(&user1Participant).Error; err != nil {
			t.Fatalf("Failed to add user1 to group DM: %v", err)
		}
		if err := db.Postgresql.Create(&user2Participant).Error; err != nil {
			t.Fatalf("Failed to add user2 to group DM: %v", err)
		}

		// Create Thread with Media in Elasticsearch for Group DM
		threadID := utility.GenerateUUID()
		mediaID := utility.GenerateUUID()
		mediaFiles := []map[string]any{
			{
				"id":        mediaID,
				"file_name": "group_dm_file.png",
				"file_type": "png",
				"mime_type": "image/png",
				"file_link": "http://example.com/group_dm_file.png",
			},
		}

		thread := map[string]any{
			"id":          threadID,
			"thread_id":   threadID,
			"channels_id": groupDmChannelID,
			"user_id":     user1.ID, // User 1 sends file
			"org_id":      org.ID,
			"message":     "Test Group DM message with file",
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
			"media":       mediaFiles,
		}

		// Index and wait
		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}
		time.Sleep(2 * time.Second)

		// Test User 2 (participant) fetching the file
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/media", org.ID, groupDmChannelID), nil)
		// We need User 2 token
		token2 := tst.GetLoginToken(t, gin.Default(), authController, models.LoginRequestModel{Email: user2SignUpData.Email, Password: user2SignUpData.Password})
		req.Header.Set("Authorization", "Bearer "+token2)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].([]interface{})
		if !ok || len(data) == 0 {
			t.Fatal("Expected files in response data")
		}

		found := false
		for _, item := range data {
			file := item.(map[string]interface{})
			if file["id"] == mediaID {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected file ID %s not found in response", mediaID)
		}
	})
}
