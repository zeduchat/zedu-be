package test_channel

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
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetChannelFiles(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create User
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser_cf_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Test",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("testuser_cf_%v", currUUID),
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

	// Create Channel
	channelID := utility.GenerateUUID()
	testChannel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("test-channel-file-%v", currUUID),
		Description:    "Test Channel for files",
		OwnerId:        user.ID,
		OrganisationID: org.ID,
	}
	if err := db.Postgresql.Create(&testChannel).Error; err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Add user to channel
	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user.ID,
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	// Create Thread with Media in Elasticsearch
	threadID := utility.GenerateUUID()
	mediaID := utility.GenerateUUID()
	mediaFile := map[string]interface{}{
		"id":        mediaID,
		"file_name": "test_channel_file.jpg",
		"file_type": "jpg",
		"mime_type": "image/jpeg",
		"file_link": "http://example.com/test_channel_file.jpg",
	}

	thread := map[string]interface{}{
		"thread_id":   threadID,
		"channels_id": channelID,
		"user_id":     user.ID,
		"org_id":      org.ID,
		"message":     "Test message with file",
		"created_at":  time.Now().Format(time.RFC3339),
		"media":       []interface{}{mediaFile},
	}

	if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
		t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
	}

	// Give ES time to index
	time.Sleep(2 * time.Second)

	// Setup Router with Channel Controller
	channelController := channel.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r.GET("/api/v1/channels/:channelId/files", middleware.Authorize(db.Postgresql), channelController.GetChannelFiles)

	t.Run("Get Channel Files", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/files?page=1&limit=10", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+token)

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
		if !ok {
			t.Fatal("Response data is not an array")
		}

		if len(data) == 0 {
			t.Fatal("Expected files in response")
		}

		found := false
		for _, item := range data {
			threadItem, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if threadItem["thread_id"] == threadID {
				if threadItem["username"] == nil || threadItem["avatar_url"] == nil {
					t.Errorf("Profile fields missing in thread item: %v", threadItem)
				}
				mediaList, ok := threadItem["media"].([]interface{})
				if ok {
					for _, m := range mediaList {
						fileMap, ok := m.(map[string]interface{})
						if ok && fileMap["id"] == mediaID {
							found = true
							break
						}
					}
				}
			}
		}

		if !found {
			t.Errorf("Expected file ID %s in thread %s not found in response", mediaID, threadID)
		}

		pagination, ok := response["pagination"].(map[string]interface{})
		if !ok {
			t.Fatal("Response pagination metadata is missing")
		}

		if pagination["current_page"] == nil || pagination["total_items"] == nil {
			t.Errorf("Incomplete pagination metadata in response: %v", pagination)
		}
	})

	t.Run("Get DM Channel Files and Type=Document Filter with docx", func(t *testing.T) {
		dmChannelID := utility.GenerateUUID()
		dmChannel := models.DmChannels{
			ID:          utility.GenerateUUID(),
			ChannelId:   dmChannelID,
			UserId:      user.ID,
			OrgId:       org.ID,
			ChatType:    "user",
			ChannelType: "dm",
		}
		if err := db.Postgresql.Create(&dmChannel).Error; err != nil {
			t.Fatalf("Failed to create DM channel: %v", err)
		}

		dmThreadID := utility.GenerateUUID()
		docxMediaID := utility.GenerateUUID()
		docxFile := map[string]interface{}{
			"id":        docxMediaID,
			"file_name": "sample_document.docx",
			"file_type": "docx",
			"mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"file_link": "http://example.com/sample_document.docx",
		}
		dmThread := map[string]interface{}{
			"thread_id":   dmThreadID,
			"channels_id": dmChannelID,
			"user_id":     user.ID,
			"org_id":      org.ID,
			"message":     "Test message with docx file",
			"created_at":  time.Now().Format(time.RFC3339),
			"media":       []interface{}{docxFile},
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, dmThreadID, dmThread, logger); err != nil {
			t.Fatalf("Failed to add DM thread to Elasticsearch: %v", err)
		}

		time.Sleep(2 * time.Second)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/files?type=document", dmChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for DM channel files, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].([]interface{})
		if !ok || len(data) == 0 {
			t.Fatalf("Expected docx file in DM channel response, got: %v", response)
		}
	})
}
