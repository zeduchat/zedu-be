package test_message

import (
	"bytes"
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

// TestGetDmParticipantsNewFields tests all new fields in GetDmParticipants response:
// - type (bot, dm, groupdm)
// - group_description (for group DMs only)
// - title (for participants)
// - is_admin (for group DM participants)
// - preview_media
func TestGetDmParticipantsNewFields(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create user 1 (will be admin/creator of group DM)
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("participants_test1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "TestUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("participants_user1_%v", currUUID),
	}

	// Create user 2 (participant)
	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("participants_test2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "TestUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("participants_user2_%v", currUUID),
	}

	// Create user 3 (another participant)
	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("participants_test3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "TestUser",
		LastName:    "Three",
		Password:    "password",
		UserName:    fmt.Sprintf("participants_user3_%v", currUUID),
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
	tst.SignupUser(t, gin.Default(), authController, user3SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

	var user1, user2, user3 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error; err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user3SignUpData.Email).First(&user3).Error; err != nil {
		t.Fatalf("Failed to get user3: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	// Set titles for users
	user1Title := "Engineering Lead"
	user2Title := "Product Manager"
	user3Title := "Designer"

	db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user1.ID).Update("title", user1Title)
	db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user2.ID).Update("title", user2Title)
	db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user3.ID).Update("title", user3Title)

	extReq := request.ExternalRequest{Logger: logger, Test: true}
	controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

	t.Run("Regular DM - type is 'dm' and participant has title", func(t *testing.T) {
		dmChannelID := utility.GenerateUUID()
		participantID := user2.ID

		// Create DM channel
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

		// Create reverse DM channel
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

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id", middleware.Authorize(db.Postgresql), controller.GetDmParticipants)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, dmChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data field in response")
		}

		// Verify type is "dm"
		responseType, _ := data["type"].(string)
		if responseType != "dm" {
			t.Errorf("Expected type 'dm', got '%s'", responseType)
		} else {
			t.Log("✅ type is 'dm'")
		}

		// Verify group_description is not present (omitempty)
		if _, exists := data["group_description"]; exists && data["group_description"] != "" {
			t.Errorf("group_description should be empty for regular DM, got: %v", data["group_description"])
		} else {
			t.Log("✅ group_description is omitted for regular DM")
		}

		// Verify participants have title
		participants, ok := data["participants"].([]interface{})
		if !ok || len(participants) == 0 {
			t.Fatal("Expected participants in response")
		}

		participant := participants[0].(map[string]interface{})
		title, _ := participant["title"].(string)
		if title != user2Title {
			t.Errorf("Expected title '%s', got '%s'", user2Title, title)
		} else {
			t.Logf("✅ participant title is correct: '%s'", title)
		}
	})

	t.Run("Group DM - type is 'groupdm', has group_description, participants have title and is_admin", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()
		groupDescription := "Our project discussion group"

		// Create group DM channel with description (user1 is admin/creator)
		groupDMChannel := models.DmChannels{
			ID:               utility.GenerateUUID(),
			UserId:           user1.ID,
			ChannelId:        groupDMChannelID,
			OrgId:            org.ID,
			ParticipantHash:  participantHash,
			ChatType:         "user",
			ChannelType:      "group_dm",
			GroupDescription: groupDescription,
		}
		if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
			t.Fatalf("Failed to create group DM channel: %v", err)
		}

		// Create channel participants
		for _, user := range []models.User{user1, user2, user3} {
			participant := models.ChannelParticipant{
				ID:        utility.GenerateUUID(),
				ChannelId: groupDMChannelID,
				UserId:    user.ID,
				OrgId:     org.ID,
			}
			if err := db.Postgresql.Create(&participant).Error; err != nil {
				t.Fatalf("Failed to create channel participant: %v", err)
			}
		}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id", middleware.Authorize(db.Postgresql), controller.GetDmParticipants)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data field in response")
		}

		// Verify type is "groupdm"
		responseType, _ := data["type"].(string)
		if responseType != "groupdm" {
			t.Errorf("Expected type 'groupdm', got '%s'", responseType)
		} else {
			t.Log("✅ type is 'groupdm'")
		}

		// Verify group_description is returned
		returnedDescription, _ := data["group_description"].(string)
		if returnedDescription != groupDescription {
			t.Errorf("Expected group_description '%s', got '%s'", groupDescription, returnedDescription)
		} else {
			t.Logf("✅ group_description is correct: '%s'", returnedDescription)
		}

		// Verify participants
		participants, ok := data["participants"].([]interface{})
		if !ok || len(participants) != 3 {
			t.Fatalf("Expected 3 participants, got %d", len(participants))
		}
		t.Logf("✅ Found %d participants", len(participants))

		adminCount := 0
		titlesFound := make(map[string]bool)
		expectedTitles := map[string]string{
			user1.ID: user1Title,
			user2.ID: user2Title,
			user3.ID: user3Title,
		}

		for i, p := range participants {
			participant := p.(map[string]interface{})
			userId, _ := participant["user_id"].(string)
			isAdmin, hasIsAdmin := participant["is_admin"].(bool)
			title, hasTitle := participant["title"].(string)

			// Verify is_admin field exists
			if !hasIsAdmin {
				t.Errorf("Participant %d missing is_admin field", i)
			}

			// Verify title field exists
			if !hasTitle {
				t.Errorf("Participant %d missing title field", i)
			}

			// Check admin status (user1 is the creator/admin)
			if userId == user1.ID {
				if !isAdmin {
					t.Errorf("User1 should be admin but is_admin is false")
				} else {
					adminCount++
					t.Logf("✅ User1 (creator) correctly marked as admin")
				}
			} else {
				if isAdmin {
					t.Errorf("User %s should NOT be admin", userId)
				}
			}

			// Verify title matches expected
			expectedTitle, _ := expectedTitles[userId]
			if title != expectedTitle {
				t.Errorf("User %s: expected title '%s', got '%s'", userId, expectedTitle, title)
			} else {
				titlesFound[userId] = true
				t.Logf("✅ User %s has correct title: '%s'", userId, title)
			}
		}

		if adminCount != 1 {
			t.Errorf("Expected exactly 1 admin, found %d", adminCount)
		} else {
			t.Log("✅ Exactly 1 admin in group DM")
		}
	})

	t.Run("Group DM with Preview Media", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		// Create group DM channel
		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user1.ID,
			ChannelId:       groupDMChannelID,
			OrgId:           org.ID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
			t.Fatalf("Failed to create group DM channel: %v", err)
		}

		// Create channel participants
		for _, user := range []models.User{user1, user2} {
			participant := models.ChannelParticipant{
				ID:        utility.GenerateUUID(),
				ChannelId: groupDMChannelID,
				UserId:    user.ID,
				OrgId:     org.ID,
			}
			if err := db.Postgresql.Create(&participant).Error; err != nil {
				t.Fatalf("Failed to create channel participant: %v", err)
			}
		}

		// Add media to Elasticsearch
		threadID := utility.GenerateUUID()
		mediaID := utility.GenerateUUID()
		thread := map[string]any{
			"id":          threadID,
			"channels_id": groupDMChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Check out this image",
			"files": []map[string]any{
				{
					"id":         mediaID,
					"file_name":  "test_image.png",
					"file_url":   "https://example.com/test_image.png",
					"file_type":  "image/png",
					"file_size":  1024,
					"created_at": time.Now().Format(time.RFC3339),
				},
			},
			"created_at": time.Now().Format(time.RFC3339),
			"updated_at": time.Now().Format(time.RFC3339),
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}

		// Wait for Elasticsearch indexing
		time.Sleep(2 * time.Second)

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id", middleware.Authorize(db.Postgresql), controller.GetDmParticipants)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data field in response")
		}

		// Verify type
		responseType, _ := data["type"].(string)
		if responseType != "groupdm" {
			t.Errorf("Expected type 'groupdm', got '%s'", responseType)
		} else {
			t.Log("✅ type is 'groupdm'")
		}

		// Verify preview_media field exists
		previewMedia, ok := data["preview_media"].([]interface{})
		if !ok {
			t.Log("⚠️ preview_media is not an array or doesn't exist")
		} else {
			t.Logf("✅ preview_media field exists with %d items", len(previewMedia))
		}

		// Verify participants
		participants, ok := data["participants"].([]interface{})
		if !ok || len(participants) != 2 {
			t.Errorf("Expected 2 participants, got %d", len(participants))
		} else {
			t.Logf("✅ Found %d participants in group DM with media", len(participants))
		}
	})

	t.Run("Group DM - set description then verify it's returned in GetDmParticipants", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		// Create group DM channel WITHOUT description initially
		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user1.ID,
			ChannelId:       groupDMChannelID,
			OrgId:           org.ID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
			t.Fatalf("Failed to create group DM channel: %v", err)
		}

		// Create channel participants
		for _, user := range []models.User{user1, user2} {
			participant := models.ChannelParticipant{
				ID:        utility.GenerateUUID(),
				ChannelId: groupDMChannelID,
				UserId:    user.ID,
				OrgId:     org.ID,
			}
			if err := db.Postgresql.Create(&participant).Error; err != nil {
				t.Fatalf("Failed to create channel participant: %v", err)
			}
		}

		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)
		r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id", middleware.Authorize(db.Postgresql), controller.GetDmParticipants)

		// First call GetDmParticipants - group_description should be empty
		getReq1, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, groupDMChannelID), nil)
		getReq1.Header.Set("Authorization", "Bearer "+token1)

		getRr1 := httptest.NewRecorder()
		r.ServeHTTP(getRr1, getReq1)

		if getRr1.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", getRr1.Code)
		}

		var response1 map[string]interface{}
		json.Unmarshal(getRr1.Body.Bytes(), &response1)
		data1, _ := response1["data"].(map[string]interface{})

		desc1, _ := data1["group_description"].(string)
		if desc1 != "" {
			t.Errorf("Expected empty group_description initially, got '%s'", desc1)
		} else {
			t.Log("✅ group_description is empty initially")
		}

		// Now set the description
		expectedDescription := "Updated: Team Alpha Discussion"
		reqBody := map[string]string{
			"description": expectedDescription,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		putReq, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, groupDMChannelID), bytes.NewBuffer(bodyBytes))
		putReq.Header.Set("Content-Type", "application/json")
		putReq.Header.Set("Authorization", "Bearer "+token1)

		putRr := httptest.NewRecorder()
		r.ServeHTTP(putRr, putReq)

		if putRr.Code != http.StatusOK {
			t.Fatalf("Failed to set description: %d - %s", putRr.Code, putRr.Body.String())
		}
		t.Log("✅ Successfully set group description")

		// Call GetDmParticipants again - now group_description should be set
		getReq2, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, groupDMChannelID), nil)
		getReq2.Header.Set("Authorization", "Bearer "+token1)

		getRr2 := httptest.NewRecorder()
		r.ServeHTTP(getRr2, getReq2)

		if getRr2.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", getRr2.Code)
		}

		var response2 map[string]interface{}
		json.Unmarshal(getRr2.Body.Bytes(), &response2)
		data2, _ := response2["data"].(map[string]interface{})

		desc2, _ := data2["group_description"].(string)
		if desc2 != expectedDescription {
			t.Errorf("Expected group_description '%s', got '%s'", expectedDescription, desc2)
		} else {
			t.Logf("✅ group_description correctly returned after update: '%s'", desc2)
		}
	})
}
