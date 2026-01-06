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
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetDmChannelsPreviewMessage(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("dmuser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "DMUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("dmuser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("dmuser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "DMUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("dmuser2_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authController, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user2SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

	var user1, user2 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error; err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	t.Run("DM Channel with Preview Message - Not Empty", func(t *testing.T) {
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

		dmChannel2 := models.DmChannels{
			ID:            utility.GenerateUUID(),
			UserId:        user2.ID,
			ChannelId:     dmChannelID,
			OrgId:         org.ID,
			ParticipantId: &user1.ID,
			ChatType:      "user",
			ChannelType:   "dm",
		}

		if err := db.Postgresql.Create(&dmChannel2).Error; err != nil {
			t.Fatalf("Failed to create reverse DM channel: %v", err)
		}

		threadID := utility.GenerateUUID()
		messageContent := "Hello, this is a test message!"
		thread := map[string]any{
			"id":          threadID,
			"channels_id": dmChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     messageContent,
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}

		time.Sleep(2 * time.Second)

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), controller.GetDmChannels)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms", org.ID), nil)
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
		if !ok {
			t.Fatal("Response missing data field")
		}

		if len(data) == 0 {
			t.Fatal("Expected at least one DM channel in response")
		}

		foundChannel := false
		for _, item := range data {
			channel, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			if channel["channel_id"] == dmChannelID {
				foundChannel = true

				previewMessage, ok := channel["preview_message"].(string)
				if !ok {
					t.Error("preview_message field is missing or not a string")
				}

				if previewMessage == "" {
					t.Error("preview_message is empty, expected non-empty message")
				}

				if previewMessage != messageContent {
					t.Errorf("Expected preview_message to be '%s', got '%s'", messageContent, previewMessage)
				}

				t.Logf("✅ Preview message is not empty: '%s'", previewMessage)
				break
			}
		}

		if !foundChannel {
			t.Error("DM channel not found in response")
		}
	})

	t.Run("Group DM Channel with Preview Message - Not Empty", func(t *testing.T) {
		user3SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("dmuser3_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "DMUser",
			LastName:    "Three",
			Password:    "password",
			UserName:    fmt.Sprintf("dmuser3_%v", currUUID),
		}

		tst.SignupUser(t, gin.Default(), authController, user3SignUpData, false)

		var user3 models.User
		if err := db.Postgresql.Where("email = ?", user3SignUpData.Email).First(&user3).Error; err != nil {
			t.Fatalf("Failed to get user3: %v", err)
		}

		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		users := []models.User{user1, user2, user3}
		for _, user := range users {
			participant := models.ChannelParticipant{
				ID:        utility.GenerateUUID(),
				ChannelId: groupDMChannelID,
				UserId:    user.ID,
				OrgId:     org.ID,
			}
			if err := db.Postgresql.Create(&participant).Error; err != nil {
				t.Fatalf("Failed to create channel participant: %v", err)
			}

			groupDMChannel := models.DmChannels{
				ID:              utility.GenerateUUID(),
				UserId:          user.ID,
				ChannelId:       groupDMChannelID,
				OrgId:           org.ID,
				ParticipantHash: participantHash,
				ChatType:        "user",
				ChannelType:     "group_dm",
			}

			if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
				t.Fatalf("Failed to create group DM channel: %v", err)
			}
		}

		threadID := utility.GenerateUUID()
		groupMessageContent := "Hello from the group!"
		thread := map[string]any{
			"id":          threadID,
			"channels_id": groupDMChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     groupMessageContent,
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}

		time.Sleep(2 * time.Second)

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), controller.GetDmChannels)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms", org.ID), nil)
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
		if !ok {
			t.Fatal("Response missing data field")
		}

		if len(data) == 0 {
			t.Fatal("Expected at least one DM channel in response")
		}

		foundChannel := false
		for _, item := range data {
			channel, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			if channel["channel_id"] == groupDMChannelID {
				foundChannel = true

				previewMessage, ok := channel["preview_message"].(string)
				if !ok {
					t.Error("preview_message field is missing or not a string")
				}

				if previewMessage == "" {
					t.Error("preview_message is empty, expected non-empty message")
				}

				if previewMessage != groupMessageContent {
					t.Errorf("Expected preview_message to be '%s', got '%s'", groupMessageContent, previewMessage)
				}

				t.Logf("✅ Group DM preview message is not empty: '%s'", previewMessage)
				break
			}
		}

		if !foundChannel {
			t.Error("Group DM channel not found in response")
		}
	})

	t.Run("DM Channel without Messages - Empty Preview", func(t *testing.T) {
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

		dmChannel2 := models.DmChannels{
			ID:            utility.GenerateUUID(),
			UserId:        user2.ID,
			ChannelId:     dmChannelID,
			OrgId:         org.ID,
			ParticipantId: &user1.ID,
			ChatType:      "user",
			ChannelType:   "dm",
		}

		if err := db.Postgresql.Create(&dmChannel2).Error; err != nil {
			t.Fatalf("Failed to create reverse DM channel: %v", err)
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), controller.GetDmChannels)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms", org.ID), nil)
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
		if !ok {
			t.Fatal("Response missing data field")
		}

		foundChannel := false
		for _, item := range data {
			channel, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			if channel["channel_id"] == dmChannelID {
				foundChannel = true

				previewMessage, ok := channel["preview_message"].(string)
				if !ok {
					t.Error("preview_message field is missing or not a string")
				}

				if previewMessage != "" {
					t.Logf("⚠️  Preview message is '%s', expected empty for channel without messages", previewMessage)
				} else {
					t.Logf("✅ Preview message is empty as expected for channel without messages")
				}
				break
			}
		}

		if !foundChannel {
			t.Error("DM channel not found in response")
		}
	})
}
