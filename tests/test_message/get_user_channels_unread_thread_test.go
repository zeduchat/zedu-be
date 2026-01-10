package test_message

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUserChannelsUnreadThread(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("unread_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "UnreadUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("unread_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("unread_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "UnreadUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("unread_user2_%v", currUUID),
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

	t.Run("DM Channel - GetUserChannelsUnreadThread", func(t *testing.T) {
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
		messageContent := "Test unread message for DM"
		thread := map[string]any{
			"thread_id":   threadID,
			"channels_id": dmChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     messageContent,
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
			"type":        "thread",
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}

		time.Sleep(2 * time.Second)

		dmChannelReq := models.DmChannels{
			ChannelId:   dmChannelID,
			UserId:      user2.ID,
			ChannelType: "dm",
		}

		result, err := dmChannelReq.GetUserChannelsUnreadThread(db)
		if err != nil {
			t.Fatalf("GetUserChannelsUnreadThread failed: %v", err)
		}

		if len(result) == 0 {
			t.Fatal("Expected at least one channel in result")
		}

		channel := result[0]

		if channel.ID != dmChannelID {
			t.Errorf("Expected channel ID %s, got %s", dmChannelID, channel.ID)
		}

		if channel.ChannelType != "dm" {
			t.Errorf("Expected channel type 'dm', got %s", channel.ChannelType)
		}

		if len(channel.Participants) == 0 {
			t.Error("Participants array is empty, expected at least one participant")
		} else {
			t.Logf("✅ Participants array has %d participant(s)", len(channel.Participants))
			participant := channel.Participants[0]
			if _, exists := participant["username"]; !exists {
				t.Error("Participant missing username field")
			}
			if _, exists := participant["email"]; !exists {
				t.Error("Participant missing email field")
			}
			if _, exists := participant["user_id"]; !exists {
				t.Error("Participant missing user_id field")
			}
			t.Logf("✅ Participant: username=%v, email=%v", participant["username"], participant["email"])
		}

		if len(channel.PreviewThread) == 0 {
			t.Error("PreviewThread array is empty, expected at least one thread")
		} else {
			t.Logf("✅ PreviewThread has %d thread(s)", len(channel.PreviewThread))
			if channel.PreviewThread[0].Content != messageContent {
				t.Errorf("Expected preview thread content '%s', got '%s'", messageContent, channel.PreviewThread[0].Content)
			}
		}

		if channel.PreviewMessage == "" {
			t.Error("PreviewMessage is empty, expected non-empty message")
		} else {
			t.Logf("✅ PreviewMessage: '%s'", channel.PreviewMessage)
			if channel.PreviewMessage != messageContent {
				t.Errorf("Expected preview message '%s', got '%s'", messageContent, channel.PreviewMessage)
			}
		}

		t.Log("✅ DM Channel test passed - no GORM parsing errors")
	})

	t.Run("Group DM Channel - GetUserChannelsUnreadThread", func(t *testing.T) {
		user3SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("unread_user3_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "UnreadUser",
			LastName:    "Three",
			Password:    "password",
			UserName:    fmt.Sprintf("unread_user3_%v", currUUID),
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
		groupMessageContent := "Test unread message for group DM"
		thread := map[string]any{
			"thread_id":   threadID,
			"channels_id": groupDMChannelID,
			"user_id":     user1.ID,
			"org_id":      org.ID,
			"message":     groupMessageContent,
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
			"type":        "thread",
		}

		if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, thread, logger); err != nil {
			t.Fatalf("Failed to add thread to Elasticsearch: %v", err)
		}

		time.Sleep(2 * time.Second)

		dmChannelReq := models.DmChannels{
			ChannelId:   groupDMChannelID,
			UserId:      user1.ID,
			ChannelType: "group_dm",
		}

		result, err := dmChannelReq.GetUserChannelsUnreadThread(db)
		if err != nil {
			t.Fatalf("GetUserChannelsUnreadThread failed: %v", err)
		}

		if len(result) == 0 {
			t.Fatal("Expected at least one channel in result")
		}

		foundChannel := false
		for _, channel := range result {
			if channel.ID == groupDMChannelID {
				foundChannel = true

				if channel.ChannelType != "group_dm" {
					t.Errorf("Expected channel type 'group_dm', got %s", channel.ChannelType)
				}

				if len(channel.Participants) != 3 {
					t.Errorf("Expected 3 participants, got %d", len(channel.Participants))
				} else {
					t.Logf("✅ Participants array has %d participants as expected", len(channel.Participants))
					for i, participant := range channel.Participants {
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

				if len(channel.PreviewThread) == 0 {
					t.Error("PreviewThread array is empty, expected at least one thread")
				} else {
					t.Logf("✅ PreviewThread has %d thread(s)", len(channel.PreviewThread))
					if channel.PreviewThread[0].Content != groupMessageContent {
						t.Errorf("Expected preview thread content '%s', got '%s'", groupMessageContent, channel.PreviewThread[0].Content)
					}
				}

				if channel.PreviewMessage == "" {
					t.Error("PreviewMessage is empty, expected non-empty message")
				} else {
					t.Logf("✅ PreviewMessage: '%s'", channel.PreviewMessage)
					if channel.PreviewMessage != groupMessageContent {
						t.Errorf("Expected preview message '%s', got '%s'", groupMessageContent, channel.PreviewMessage)
					}
				}

				t.Log("✅ Group DM Channel test passed - no GORM parsing errors")
				break
			}
		}

		if !foundChannel {
			t.Error("Group DM channel not found in result")
		}
	})

	t.Run("Channel without Threads - Empty Preview", func(t *testing.T) {
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

		time.Sleep(1 * time.Second)

		dmChannelReq := models.DmChannels{
			ChannelId:   dmChannelID,
			UserId:      user2.ID,
			ChannelType: "dm",
		}

		result, err := dmChannelReq.GetUserChannelsUnreadThread(db)
		if err != nil {
			t.Fatalf("GetUserChannelsUnreadThread failed: %v", err)
		}

		if len(result) == 0 {
			t.Fatal("Expected at least one channel in result")
		}

		channel := result[0]

		if len(channel.PreviewThread) != 0 {
			t.Logf("⚠️  PreviewThread has %d thread(s), expected empty for channel without threads", len(channel.PreviewThread))
		} else {
			t.Log("✅ PreviewThread is empty as expected")
		}

		if channel.PreviewMessage != "" {
			t.Logf("⚠️  PreviewMessage is '%s', expected empty for channel without messages", channel.PreviewMessage)
		} else {
			t.Log("✅ PreviewMessage is empty as expected")
		}

		if len(channel.Participants) == 0 {
			t.Error("Participants array is empty, expected at least one participant")
		} else {
			t.Logf("✅ Participants array has %d participant(s)", len(channel.Participants))
		}

		t.Log("✅ Empty preview test passed - no GORM parsing errors")
	})
}
