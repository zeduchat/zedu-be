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

				previewThread, ok := channel["preview_thread"].([]interface{})
				if !ok {
					t.Error("preview_thread field is missing or not an array")
				} else {
					if len(previewThread) == 0 {
						t.Error("preview_thread array is empty, expected at least one thread")
					} else {
						t.Logf("✅ preview_thread is an array with %d thread(s)", len(previewThread))
						firstThread, ok := previewThread[0].(map[string]interface{})
						if !ok {
							t.Error("First thread in preview_thread is not a valid object")
						} else {
							if threadMsg, exists := firstThread["message"]; exists {
								t.Logf("✅ First thread message: '%v'", threadMsg)
							}
							if threadID, exists := firstThread["thread_id"]; exists {
								t.Logf("✅ First thread ID: '%v'", threadID)
							}
						}
					}
				}

				participants, ok := channel["participants"].([]interface{})
				if !ok {
					t.Error("participants field is missing or not an array")
				} else {
					if len(participants) == 0 {
						t.Error("participants array is empty, expected at least one participant")
					} else {
						t.Logf("✅ participants array has %d participant(s)", len(participants))
						for i, p := range participants {
							participant, ok := p.(map[string]interface{})
							if !ok {
								t.Errorf("Participant %d is not a valid object", i)
								continue
							}
							if _, exists := participant["username"]; !exists {
								t.Errorf("Participant %d missing username field", i)
							}
							if _, exists := participant["email"]; !exists {
								t.Errorf("Participant %d missing email field", i)
							}
							if _, exists := participant["user_id"]; !exists {
								t.Errorf("Participant %d missing user_id field", i)
							}
							t.Logf("✅ Participant %d: username=%v, email=%v", i, participant["username"], participant["email"])
						}
					}
				}
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

				previewThread, ok := channel["preview_thread"].([]interface{})
				if !ok {
					t.Error("preview_thread field is missing or not an array")
				} else {
					if len(previewThread) == 0 {
						t.Error("preview_thread array is empty, expected at least one thread")
					} else {
						t.Logf("✅ preview_thread is an array with %d thread(s)", len(previewThread))
					}
				}

				participants, ok := channel["participants"].([]interface{})
				if !ok {
					t.Error("participants field is missing or not an array")
				} else {
					if len(participants) != 3 {
						t.Errorf("Expected 3 participants in group DM, got %d", len(participants))
					} else {
						t.Logf("✅ participants array has %d participants as expected", len(participants))
						for i, p := range participants {
							participant, ok := p.(map[string]interface{})
							if !ok {
								t.Errorf("Participant %d is not a valid object", i)
								continue
							}
							if _, exists := participant["username"]; !exists {
								t.Errorf("Participant %d missing username field", i)
							}
							if _, exists := participant["email"]; !exists {
								t.Errorf("Participant %d missing email field", i)
							}
							t.Logf("✅ Participant %d: username=%v, email=%v", i, participant["username"], participant["email"])
						}
					}
				}
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

				previewThread, ok := channel["preview_thread"].([]interface{})
				if !ok {
					t.Error("preview_thread field is missing or not an array")
				} else {
					if len(previewThread) == 0 {
						t.Logf("✅ preview_thread array is empty as expected for channel without threads")
					} else {
						t.Logf("⚠️  preview_thread has %d thread(s), expected empty for channel without messages", len(previewThread))
					}
				}

				participants, ok := channel["participants"].([]interface{})
				if !ok {
					t.Error("participants field is missing or not an array")
				} else {
					if len(participants) == 0 {
						t.Error("participants array is empty, expected at least one participant")
					} else {
						t.Logf("✅ participants array has %d participant(s)", len(participants))
					}
				}
				break
			}
		}

		if !foundChannel {
			t.Error("DM channel not found in response")
		}
	})
}

// TestGetDmChannelsAdminAndTitle tests that the is_admin and title fields are correctly returned
func TestGetDmChannelsAdminAndTitle(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create user 1 (will be admin/creator)
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("admin_test_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Admin",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("admin_user_%v", currUUID),
	}

	// Create user 2 (will be a regular participant)
	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("member_test_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Member",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("member_user_%v", currUUID),
	}

	// Create user 3
	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("member2_test_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Member2",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("member2_user_%v", currUUID),
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

	// Update user profiles with titles
	user1Title := "Senior Engineer"
	user2Title := "Product Manager"
	user3Title := "Designer"

	db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user1.ID).Update("title", user1Title)
	db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user2.ID).Update("title", user2Title)
	db.Postgresql.Model(&models.Profile{}).Where("userid = ?", user3.ID).Update("title", user3Title)

	t.Run("Group DM - Admin and Title Fields", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		// Create the group DM channel with user1 as the creator (admin)
		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user1.ID, // User1 is the creator/admin
			ChannelId:       groupDMChannelID,
			OrgId:           org.ID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
			t.Fatalf("Failed to create group DM channel: %v", err)
		}

		// Create channel participants for all users
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

			if channel["channel_id"] == groupDMChannelID {
				foundChannel = true

				participants, ok := channel["participants"].([]interface{})
				if !ok {
					t.Error("participants field is missing or not an array")
					break
				}

				t.Logf("✅ Found %d participants in group DM", len(participants))

				adminCount := 0
				for i, p := range participants {
					participant, ok := p.(map[string]interface{})
					if !ok {
						t.Errorf("Participant %d is not a valid object", i)
						continue
					}

					userId, _ := participant["user_id"].(string)
					isAdmin, hasIsAdmin := participant["is_admin"].(bool)
					title, hasTitle := participant["title"].(string)

					// Check is_admin field exists
					if !hasIsAdmin {
						t.Errorf("Participant %d missing is_admin field", i)
					}

					// Check title field exists
					if !hasTitle {
						t.Errorf("Participant %d missing title field", i)
					}

					// Verify admin is correctly set (user1 created the channel)
					switch userId {
					case user1.ID:
						if !isAdmin {
							t.Errorf("User1 should be admin but is_admin is false")
						} else {
							adminCount++
							t.Logf("✅ User1 (creator) correctly marked as admin")
						}
						if title != user1Title {
							t.Errorf("User1 title mismatch: expected '%s', got '%s'", user1Title, title)
						} else {
							t.Logf("✅ User1 title correctly set: '%s'", title)
						}
					case user2.ID:
						if isAdmin {
							t.Errorf("User2 should NOT be admin")
						} else {
							t.Logf("✅ User2 correctly NOT marked as admin")
						}
						if title != user2Title {
							t.Errorf("User2 title mismatch: expected '%s', got '%s'", user2Title, title)
						} else {
							t.Logf("✅ User2 title correctly set: '%s'", title)
						}
					case user3.ID:
						if isAdmin {
							t.Errorf("User3 should NOT be admin")
						} else {
							t.Logf("✅ User3 correctly NOT marked as admin")
						}
						if title != user3Title {
							t.Errorf("User3 title mismatch: expected '%s', got '%s'", user3Title, title)
						} else {
							t.Logf("✅ User3 title correctly set: '%s'", title)
						}
					}

					t.Logf("  Participant %d: user_id=%s, is_admin=%v, title='%s'", i, userId, isAdmin, title)
				}

				if adminCount != 1 {
					t.Errorf("Expected exac tly 1 admin, found %d", adminCount)
				}

				break
			}
		}

		if !foundChannel {
			t.Error("Group DM channel not found in response")
		}
	})

	t.Run("Group DM - Empty Title when Profile has no Title", func(t *testing.T) {
		// Create a user without a title
		user4SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("notitle_test_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "NoTitle",
			LastName:    "User",
			Password:    "password",
			UserName:    fmt.Sprintf("notitle_user_%v", currUUID),
		}
		tst.SignupUser(t, gin.Default(), authController, user4SignUpData, false)

		var user4 models.User
		if err := db.Postgresql.Where("email = ?", user4SignUpData.Email).First(&user4).Error; err != nil {
			t.Fatalf("Failed to get user4: %v", err)
		}

		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

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

		// Add participants
		for _, user := range []models.User{user1, user4} {
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

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), controller.GetDmChannels)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Fatal("Response missing data field")
		}

		for _, item := range data {
			channel, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			if channel["channel_id"] == groupDMChannelID {
				participants, ok := channel["participants"].([]interface{})
				if !ok {
					t.Error("participants field is missing")
					break
				}

				for _, p := range participants {
					participant, ok := p.(map[string]interface{})
					if !ok {
						continue
					}

					userId, _ := participant["user_id"].(string)
					title, hasTitle := participant["title"].(string)

					if userId == user4.ID {
						if !hasTitle {
							t.Error("title field should exist even when empty")
						} else if title != "" {
							t.Errorf("Expected empty title for user4, got '%s'", title)
						} else {
							t.Logf("✅ User4 (no profile title) correctly has empty title field")
						}
					}
				}
				break
			}
		}
	})
}

// TestGetDmChannelMedia tests the GET /organisations/:org_id/dms/:channel_id/media endpoint
func TestGetDmChannelMedia(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create test users
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("media_test1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Media",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("media_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("media_test2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Media",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("media_user2_%v", currUUID),
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

	t.Run("Get Media - Success with Media", func(t *testing.T) {
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

		// Create reverse channel entry
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

		// Create a thread with media in Elasticsearch
		threadID := utility.GenerateUUID()
		mediaFiles := []map[string]any{
			{
				"id":        utility.GenerateUUID(),
				"file_name": "test_image.jpg",
				"file_type": "jpg",
				"mime_type": "image/jpeg",
				"file_link": "https://example.com/test_image.jpg",
			},
			{
				"id":        utility.GenerateUUID(),
				"file_name": "test_video.mp4",
				"file_type": "mp4",
				"mime_type": "video/mp4",
				"file_link": "https://example.com/test_video.mp4",
			},
		}

		thread := map[string]any{
			"id":          threadID,
			"channels_id": dmChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Test message with media",
			"media":       mediaFiles,
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}

		// Wait for Elasticsearch to index
		time.Sleep(2 * time.Second)

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/:channel_id/media", middleware.Authorize(db.Postgresql), controller.GetDmChannelMedia)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/media", org.ID, dmChannelID), nil)
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

		t.Logf("✅ Got successful response")

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Error("Response missing data field or data is not an array")
			return
		}

		t.Logf("✅ Found %d media files", len(data))

		// Check pagination exists
		pagination, hasPagination := response["pagination"].(map[string]interface{})
		if hasPagination {
			t.Logf("✅ Pagination: current_page=%v, total_items=%v", pagination["current_page"], pagination["total_items"])
		}
	})

	t.Run("Get Media - Empty Channel", func(t *testing.T) {
		dmChannelID := utility.GenerateUUID()
		participantID := user2.ID

		// Create DM channel without any messages/media
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
		r.GET("/api/v1/organisations/:org_id/dms/:channel_id/media", middleware.Authorize(db.Postgresql), controller.GetDmChannelMedia)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/media", org.ID, dmChannelID), nil)
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

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Error("Response missing data field")
			return
		}

		if len(data) == 0 {
			t.Logf("✅ Empty media array returned for channel with no media")
		} else {
			t.Logf("⚠️ Expected empty array, got %d items", len(data))
		}
	})

	t.Run("Get Media - Invalid Channel ID", func(t *testing.T) {
		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/:channel_id/media", middleware.Authorize(db.Postgresql), controller.GetDmChannelMedia)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/invalid-uuid/media", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusBadRequest {
			t.Logf("✅ Got expected 400 Bad Request for invalid channel ID")
		} else {
			t.Logf("Got status %d for invalid channel ID", rr.Code)
		}
	})

	t.Run("Get Media - Unauthorized", func(t *testing.T) {
		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/:channel_id/media", middleware.Authorize(db.Postgresql), controller.GetDmChannelMedia)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/media", org.ID, utility.GenerateUUID()), nil)
		// No Authorization header

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusUnauthorized {
			t.Logf("✅ Got expected 401 Unauthorized")
		} else {
			t.Logf("Got status %d (expected 401)", rr.Code)
		}
	})
}

// TestGetDmParticipantsWithPreviewMedia tests that preview media is always returned
func TestGetDmParticipantsWithPreviewMedia(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create test users
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("preview_media1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Preview",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("preview_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("preview_media2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Preview",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("preview_user2_%v", currUUID),
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

	t.Run("Get Participants with preview media", func(t *testing.T) {
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

		// Create reverse channel entry
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

		// Create a thread with media in Elasticsearch
		threadID := utility.GenerateUUID()
		mediaFiles := []map[string]any{
			{
				"id":        utility.GenerateUUID(),
				"file_name": "preview_image.jpg",
				"file_type": "jpg",
				"mime_type": "image/jpeg",
				"file_link": "https://example.com/preview_image.jpg",
			},
			{
				"id":        utility.GenerateUUID(),
				"file_name": "preview_doc.pdf",
				"file_type": "pdf",
				"mime_type": "application/pdf",
				"file_link": "https://example.com/preview_doc.pdf",
			},
		}

		thread := map[string]any{
			"id":          threadID,
			"channels_id": dmChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     "Test message with media for preview",
			"media":       mediaFiles,
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id", middleware.Authorize(db.Postgresql), controller.GetDmParticipants)

		// Poll for Elasticsearch indexing with retries
		maxRetries := 10
		var response map[string]interface{}
		var previewMedia []interface{}

		for i := 0; i < maxRetries; i++ {
			time.Sleep(1 * time.Second) // Wait before each attempt

			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, dmChannelID), nil)
			req.Header.Set("Authorization", "Bearer "+token1)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				continue
			}

			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				continue
			}

			data, ok := response["data"].(map[string]interface{})
			if !ok {
				continue
			}

			if pm, ok := data["preview_media"].([]interface{}); ok && len(pm) > 0 {
				previewMedia = pm
				break
			}
		}

		if len(previewMedia) == 0 {
			t.Fatalf("Failed to retrieve preview media after %d retries", maxRetries)
		}

		t.Logf("✅ Got successful response with %d preview media files", len(previewMedia))

		data, _ := response["data"].(map[string]interface{})
		participants, ok := data["participants"].([]interface{})
		if !ok {
			t.Error("Missing participants array in response")
			return
		}
		t.Logf("✅ Found %d participants", len(participants))

		// Verify media has expected fields
		firstMedia := previewMedia[0].(map[string]interface{})
		if _, hasID := firstMedia["id"]; hasID {
			t.Logf("✅ Media has 'id' field")
		}
		if _, hasUserID := firstMedia["user_id"]; hasUserID {
			t.Logf("✅ Media has 'user_id' field")
		}
		if _, hasCreatedAt := firstMedia["created_at"]; hasCreatedAt {
			t.Logf("✅ Media has 'created_at' field")
		}
	})

	t.Run("Get Participants - preview_media is always present", func(t *testing.T) {
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
			t.Error("Response data is not an object")
			return
		}

		// Check participants exists
		participants, ok := data["participants"].([]interface{})
		if !ok {
			t.Error("Missing participants array in response")
			return
		}
		t.Logf("✅ Found %d participants", len(participants))

		// preview_media should always be present now (may be empty array)
		if _, exists := data["preview_media"]; !exists {
			t.Error("preview_media field should always be present")
		} else {
			t.Logf("✅ preview_media field is present as expected")
		}
	})
}

// TestGetDmParticipants_GroupsInCommon tests the groups_in_common feature for DM channels
func TestGetDmParticipants_GroupsInCommon(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create 3 users
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("gic_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupInCommon",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("gic_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("gic_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupInCommon",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("gic_user2_%v", currUUID),
	}

	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("gic_user3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupInCommon",
		LastName:    "User3",
		Password:    "password",
		UserName:    fmt.Sprintf("gic_user3_%v", currUUID),
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

	t.Run("DM Participants - Groups In Common Returned", func(t *testing.T) {
		// Create a GroupDM with user1, user2, and user3
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

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

		// Add all 3 users as participants
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

		// Create a DM between user1 and user2
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

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

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
			t.Fatal("Response data is not an object")
		}

		// Check type is dm
		if data["type"] != "dm" {
			t.Errorf("Expected type 'dm', got '%v'", data["type"])
		}

		participants, ok := data["participants"].([]interface{})
		if !ok || len(participants) == 0 {
			t.Fatal("Missing participants array in response")
		}

		participant := participants[0].(map[string]interface{})

		// Check groups_in_common exists and has the common group
		groupsInCommon, ok := participant["groups_in_common"].([]interface{})
		if !ok {
			t.Fatal("groups_in_common field missing or not an array")
		}

		if len(groupsInCommon) != 1 {
			t.Errorf("Expected 1 group in common, got %d", len(groupsInCommon))
		} else {
			group := groupsInCommon[0].(map[string]interface{})
			t.Logf("✅ Found group in common: channel_id=%v", group["channel_id"])

			// Verify the group has correct channel_id
			if group["channel_id"] != groupDMChannelID {
				t.Errorf("Expected channel_id '%s', got '%v'", groupDMChannelID, group["channel_id"])
			}

			// Verify participants array exists
			if groupParticipants, ok := group["participants"].([]interface{}); ok {
				t.Logf("✅ Group has %d participant names", len(groupParticipants))
			}
		}
	})

	t.Run("DM Participants - No Common Groups", func(t *testing.T) {
		// Create user4 who is NOT in any GroupDM with user1
		user4SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("gic_user4_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "GroupInCommon",
			LastName:    "User4",
			Password:    "password",
			UserName:    fmt.Sprintf("gic_user4_%v", currUUID),
		}
		tst.SignupUser(t, gin.Default(), authController, user4SignUpData, false)

		var user4 models.User
		if err := db.Postgresql.Where("email = ?", user4SignUpData.Email).First(&user4).Error; err != nil {
			t.Fatalf("Failed to get user4: %v", err)
		}

		// Create a DM between user1 and user4 (no common groups)
		dmChannelID := utility.GenerateUUID()
		participantID := user4.ID
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

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

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

		data := response["data"].(map[string]interface{})
		participants := data["participants"].([]interface{})
		participant := participants[0].(map[string]interface{})

		// groups_in_common should be empty or not present (omitempty)
		groupsInCommon, exists := participant["groups_in_common"]
		if exists {
			groups := groupsInCommon.([]interface{})
			if len(groups) != 0 {
				t.Errorf("Expected 0 groups in common, got %d", len(groups))
			} else {
				t.Logf("✅ groups_in_common is empty as expected")
			}
		} else {
			t.Logf("✅ groups_in_common field not present (omitempty working)")
		}
	})

	t.Run("GroupDM Participants - No Groups In Common Field", func(t *testing.T) {
		// Create a GroupDM
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

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

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

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

		data := response["data"].(map[string]interface{})

		// Check type is groupdm
		if data["type"] != "groupdm" {
			t.Errorf("Expected type 'groupdm', got '%v'", data["type"])
		}

		participants := data["participants"].([]interface{})
		for i, p := range participants {
			participant := p.(map[string]interface{})

			// groups_in_common should NOT be present for GroupDM participants (omitempty)
			if _, exists := participant["groups_in_common"]; exists {
				t.Errorf("Participant %d should NOT have groups_in_common field in GroupDM", i)
			} else {
				t.Logf("✅ Participant %d correctly does not have groups_in_common field", i)
			}
		}
	})

	t.Run("DM Participants - Multiple Common Groups", func(t *testing.T) {
		// Create a second GroupDM with user1 and user2
		groupDMChannelID2 := utility.GenerateUUID()
		participantHash2 := utility.GenerateUUID()

		groupDMChannel2 := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user1.ID,
			ChannelId:       groupDMChannelID2,
			OrgId:           org.ID,
			ParticipantHash: participantHash2,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		if err := db.Postgresql.Create(&groupDMChannel2).Error; err != nil {
			t.Fatalf("Failed to create second group DM channel: %v", err)
		}

		for _, user := range []models.User{user1, user2} {
			participant := models.ChannelParticipant{
				ID:        utility.GenerateUUID(),
				ChannelId: groupDMChannelID2,
				UserId:    user.ID,
				OrgId:     org.ID,
			}
			if err := db.Postgresql.Create(&participant).Error; err != nil {
				t.Fatalf("Failed to create channel participant: %v", err)
			}
		}

		// Create a new DM for this test
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

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

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

		data := response["data"].(map[string]interface{})
		participants := data["participants"].([]interface{})
		participant := participants[0].(map[string]interface{})

		groupsInCommon, ok := participant["groups_in_common"].([]interface{})
		if !ok {
			t.Fatal("groups_in_common field missing or not an array")
		}

		// Should have at least 2 common groups now (from first test + this one)
		if len(groupsInCommon) < 2 {
			t.Errorf("Expected at least 2 groups in common, got %d", len(groupsInCommon))
		} else {
			t.Logf("✅ Found %d groups in common as expected", len(groupsInCommon))
		}
	})
}
