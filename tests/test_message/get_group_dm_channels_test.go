package test_message

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetGroupDMChannels(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create 3 users
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("getgdm_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GetGDM",
		LastName:    "UserOne",
		Password:    "password",
		UserName:    fmt.Sprintf("getgdm_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("getgdm_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GetGDM",
		LastName:    "UserTwo",
		Password:    "password",
		UserName:    fmt.Sprintf("getgdm_user2_%v", currUUID),
	}

	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("getgdm_user3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GetGDM",
		LastName:    "UserThree",
		Password:    "password",
		UserName:    fmt.Sprintf("getgdm_user3_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	loginData3 := models.LoginRequestModel{
		Email:    user3SignUpData.Email,
		Password: user3SignUpData.Password,
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
	token1 := tst.GetLoginToken(t, gin.Default(), authController, loginData1)
	token3 := tst.GetLoginToken(t, gin.Default(), authController, loginData3)

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

	// Create a group DM channel with user1 and user2 as participants (user3 is NOT a participant)
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

	t.Run("Successfully Retrieve Group DM Channels for Authenticated User", func(t *testing.T) {
		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/group-dms", middleware.Authorize(db.Postgresql), controller.GetGroupDMChannels)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/group-dms", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify top-level response structure
		status, ok := response["status"].(string)
		if !ok || status != "success" {
			t.Errorf("Expected status 'success', got %v", response["status"])
		}

		statusCode, ok := response["status_code"].(float64)
		if !ok || int(statusCode) != 200 {
			t.Errorf("Expected status_code 200, got %v", response["status_code"])
		}

		message, ok := response["message"].(string)
		if !ok {
			t.Error("Response missing message field")
		}
		t.Logf("Message: %s", message)

		// Verify data is an array
		data, ok := response["data"].([]interface{})
		if !ok {
			t.Fatal("Response 'data' is not an array")
		}

		if len(data) == 0 {
			t.Fatal("Expected at least 1 group DM channel in response, got 0")
		}

		// Find our channel in the response
		var foundChannel map[string]interface{}
		for _, item := range data {
			ch, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if ch["channel_id"] == groupDMChannelID {
				foundChannel = ch
				break
			}
		}

		if foundChannel == nil {
			t.Fatalf("Expected to find channel %s in response, but it was not present", groupDMChannelID)
		}

		// Verify channel_id
		channelID, ok := foundChannel["channel_id"].(string)
		if !ok || channelID != groupDMChannelID {
			t.Errorf("Expected channel_id '%s', got '%v'", groupDMChannelID, foundChannel["channel_id"])
		}
		t.Logf("✅ channel_id present: %s", channelID)

		// Verify channel_type
		channelType, ok := foundChannel["channel_type"].(string)
		if !ok || channelType != "group_dm" {
			t.Errorf("Expected channel_type 'group_dm', got '%v'", foundChannel["channel_type"])
		}
		t.Logf("✅ channel_type present: %s", channelType)

		// Verify participants array
		participants, ok := foundChannel["participants"].([]interface{})
		if !ok {
			t.Fatal("participants field is missing or not an array")
		}

		if len(participants) != 2 {
			t.Errorf("Expected 2 participants, got %d", len(participants))
		}
		t.Logf("✅ participants count: %d", len(participants))

		// Verify participant fields match swagger docs
		for i, p := range participants {
			participant, ok := p.(map[string]interface{})
			if !ok {
				t.Errorf("Participant %d is not an object", i)
				continue
			}

			// Check required fields from swagger
			requiredFields := []string{
				"user_id", "username", "email", "avatar_url", "default_avatar_url",
			}
			for _, field := range requiredFields {
				if _, exists := participant[field]; !exists {
					t.Errorf("Participant %d missing field '%s'", i, field)
				}
			}

			// Check additional fields from the actual Participant struct
			additionalFields := []string{
				"phone", "first_name", "last_name", "full_name", "display_name",
				"title", "name_pronunciation", "timezone", "icon", "text",
				"pause_notification", "status_timeout", "workspace_id",
				"track", "links", "online", "is_active", "user_type", "is_admin",
			}
			for _, field := range additionalFields {
				if _, exists := participant[field]; !exists {
					t.Logf("⚠️  Participant %d missing additional field '%s' (not in swagger but in struct)", i, field)
				}
			}

			t.Logf("✅ Participant %d - user_id: %v, username: %v", i, participant["user_id"], participant["username"])
		}

		// Verify pagination is present
		if _, hasPagination := response["pagination"]; !hasPagination {
			t.Logf("⚠️  No pagination field in response (may be expected if wrapped differently)")
		} else {
			t.Logf("✅ pagination field present")
		}

		t.Logf("✅ GetGroupDMChannels returns correct response structure for authenticated user")
	})

	t.Run("Non-Participant User Gets No Results for This Channel", func(t *testing.T) {
		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.GET("/api/v1/organisations/:org_id/group-dms", middleware.Authorize(db.Postgresql), controller.GetGroupDMChannels)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/group-dms", org.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token3)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Fatal("Response 'data' is not an array")
		}

		// user3 is NOT a participant in any group DM, so the specific channel should not appear
		for _, item := range data {
			ch, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if ch["channel_id"] == groupDMChannelID {
				t.Errorf("Non-participant user3 should NOT see channel %s, but it appeared in the response", groupDMChannelID)
			}
		}

		t.Logf("✅ Non-participant user correctly does not see the group DM channel (total channels returned: %d)", len(data))
	})
}
