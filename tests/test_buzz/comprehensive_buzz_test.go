package test_buzz

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
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	buzzController "github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestBuzzInAllChannelTypes tests creating and joining buzz in regular channel, DM, and group DM
func TestBuzzInAllChannelTypes(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Setup controllers and router
	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	buzzCtrl := buzzController.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
	}

	// Create three test users
	users := make([]struct {
		SignupData models.CreateUserRequestModel
		Token      string
		UserID     string
	}, 3)

	for i := 0; i < 3; i++ {
		users[i].SignupData = models.CreateUserRequestModel{
			Email:       fmt.Sprintf("user%d_%s@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%d", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			Password:    "password",
			UserName:    fmt.Sprintf("user%d_%s", i, currUUID),
		}

		// Create fresh router for each signup to avoid route conflicts
		signupRouter := gin.Default()
		tst.SignupUser(t, signupRouter, authCtrl, users[i].SignupData, false)

		loginData := models.LoginRequestModel{
			Email:    users[i].SignupData.Email,
			Password: users[i].SignupData.Password,
		}
		loginRouter := gin.Default()
		users[i].Token = tst.GetLoginToken(t, loginRouter, authCtrl, loginData)

		// Extract user ID from token
		token, err := middleware.TokenValid(users[i].Token)
		if err != nil {
			t.Fatalf("Invalid token for user %d: %v", i, err)
		}
		claims := token.Claims.(jwt.MapClaims)
		if userID, ok := claims["user_id"].(string); ok {
			users[i].UserID = userID
		} else {
			t.Fatalf("Could not extract user ID from token for user %d", i)
		}
		t.Logf("✅ Created user %d: %s (ID: %s)", i, users[i].SignupData.Email, users[i].UserID)
	}

	// Create organization
	org := models.Organisation{
		ID:          utility.GenerateUUID(),
		Name:        fmt.Sprintf("TestOrg_%s", currUUID),
		Description: "Test organization for buzz",
		Email:       users[0].SignupData.Email,
		OwnerID:     users[0].UserID,
	}
	if err := db.Postgresql.Create(&org).Error; err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}
	t.Logf("✅ Created organization: %s", org.ID)

	// Setup buzz router for all tests
	r := gin.Default()

	// Test 1: Regular Channel Buzz
	t.Run("Regular Channel Buzz", func(t *testing.T) {
		t.Log("\n=== Testing Regular Channel Buzz ===")

		// Create regular channel
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           fmt.Sprintf("test-channel-%s", currUUID),
			Description:    "Test regular channel",
			OrganisationID: org.ID,
			OwnerId:        users[0].UserID,
			CreatedAt:      time.Now(),
		}
		if err := db.Postgresql.Create(&channel).Error; err != nil {
			t.Fatalf("Failed to create channel: %v", err)
		}
		t.Logf("✅ Created regular channel: %s", channelID)

		// Add both users to channel
		for i := 0; i < 2; i++ {
			userChannel := models.UserChannels{
				ChannelsID: channelID,
				UserID:     users[i].UserID,
				CreatedAt:  time.Now(),
			}
			if err := db.Postgresql.Create(&userChannel).Error; err != nil {
				t.Fatalf("Failed to add user %d to channel: %v", i, err)
			}
		}
		t.Logf("✅ Added users 0 and 1 to channel")

		// User 0 creates buzz
		createResp := createBuzz(t, r, buzzCtrl, users[0].Token, channelID)
		buzzID := createResp["buzz_id"].(string)
		t.Logf("✅ User 0 created buzz: %s", buzzID)

		// User 1 joins buzz
		joinBuzz(t, r, buzzCtrl, users[1].Token, buzzID)
		t.Logf("✅ User 1 joined buzz")

		// Verify buzz in database
		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz: %v", err)
		}
		if buzz.ChannelType != models.ChannelTypeRegular {
			t.Errorf("Expected channel_type 'channel', got '%s'", buzz.ChannelType)
		}
		t.Logf("✅ Verified buzz channel_type: %s", buzz.ChannelType)
	})

	// Test 2: DM Channel Buzz
	t.Run("DM Channel Buzz", func(t *testing.T) {
		t.Log("\n=== Testing DM Channel Buzz ===")

		// Create DM channel (User 0 and User 1)
		dmChannelID := utility.GenerateUUID()
		dmChannel := models.DmChannels{
			ID:            utility.GenerateUUID(),
			UserId:        users[0].UserID,
			ChannelId:     dmChannelID,
			OrgId:         org.ID,
			ParticipantId: &users[1].UserID,
			ChannelType:   "dm",
			ChatType:      "user",
			CreatedAt:     time.Now(),
		}
		if err := db.Postgresql.Create(&dmChannel).Error; err != nil {
			t.Fatalf("Failed to create DM channel: %v", err)
		}
		t.Logf("✅ Created DM channel: %s (User 0 -> User 1)", dmChannelID)

		// User 0 (creator) creates buzz
		createResp := createBuzz(t, r, buzzCtrl, users[0].Token, dmChannelID)
		buzzID := createResp["buzz_id"].(string)
		t.Logf("✅ User 0 (DM creator) created buzz: %s", buzzID)

		// User 1 (participant) joins buzz
		joinBuzz(t, r, buzzCtrl, users[1].Token, buzzID)
		t.Logf("✅ User 1 (DM participant) joined buzz")

		// Verify buzz in database
		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz: %v", err)
		}
		if buzz.ChannelType != models.ChannelTypeDM {
			t.Errorf("Expected channel_type 'dm_channel', got '%s'", buzz.ChannelType)
		}
		t.Logf("✅ Verified buzz channel_type: %s", buzz.ChannelType)

		// Test that participant can also create buzz
		time.Sleep(1 * time.Second) // Wait a bit
		// End previous buzz first
		endBuzzReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/end", buzzID), nil)
		endBuzzReq.Header.Set("Authorization", "Bearer "+users[0].Token)
		endBuzzResp := httptest.NewRecorder()
		endBuzzRouter := gin.Default()
		endBuzzRouter.POST("/api/v1/buzz/:id/end", middleware.Authorize(db.Postgresql), buzzCtrl.EndBuzz)
		endBuzzRouter.ServeHTTP(endBuzzResp, endBuzzReq)

		// User 1 (participant) creates new buzz in same DM
		createResp2 := createBuzz(t, r, buzzCtrl, users[1].Token, dmChannelID)
		buzzID2 := createResp2["buzz_id"].(string)
		t.Logf("✅ User 1 (DM participant) also created buzz: %s", buzzID2)
	})

	// Test 3: Group DM Buzz
	t.Run("Group DM Channel Buzz", func(t *testing.T) {
		t.Log("\n=== Testing Group DM Channel Buzz ===")

		// Create group DM channel (all 3 users)
		groupDMChannelID := utility.GenerateUUID()

		// Create DM channel record for group DM
		groupDM := models.DmChannels{
			ID:          utility.GenerateUUID(),
			UserId:      users[0].UserID,
			ChannelId:   groupDMChannelID,
			OrgId:       org.ID,
			ChannelType: "group_dm",
			ChatType:    "user",
			CreatedAt:   time.Now(),
		}
		if err := db.Postgresql.Create(&groupDM).Error; err != nil {
			t.Fatalf("Failed to create group DM: %v", err)
		}

		// Add all participants to channel_participants table
		for i := 0; i < 3; i++ {
			participant := models.ChannelParticipant{
				ID:        utility.GenerateUUID(),
				ChannelId: groupDMChannelID,
				UserId:    users[i].UserID,
				OrgId:     org.ID,
				CreatedAt: time.Now(),
			}
			if err := db.Postgresql.Create(&participant).Error; err != nil {
				t.Fatalf("Failed to add participant %d: %v", i, err)
			}
		}
		t.Logf("✅ Created group DM channel: %s (3 participants)", groupDMChannelID)

		// User 0 creates buzz
		createResp := createBuzz(t, r, buzzCtrl, users[0].Token, groupDMChannelID)
		buzzID := createResp["buzz_id"].(string)
		t.Logf("✅ User 0 created buzz in group DM: %s", buzzID)

		// User 1 and User 2 join buzz
		joinBuzz(t, r, buzzCtrl, users[1].Token, buzzID)
		t.Logf("✅ User 1 joined group DM buzz")

		joinBuzz(t, r, buzzCtrl, users[2].Token, buzzID)
		t.Logf("✅ User 2 joined group DM buzz")

		// Verify buzz in database
		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz: %v", err)
		}
		if buzz.ChannelType != models.ChannelTypeGroupDM {
			t.Errorf("Expected channel_type 'group_dm_channel', got '%s'", buzz.ChannelType)
		}
		t.Logf("✅ Verified buzz channel_type: %s", buzz.ChannelType)
	})

	t.Log("\n=== All Buzz Tests Passed ✅ ===")
}

// Helper function to create buzz
func createBuzz(t *testing.T, r *gin.Engine, ctrl buzzController.Controller, token, channelID string) map[string]interface{} {
	createBuzzData := models.CreateBuzzRequest{
		ChannelID: channelID,
	}
	bodyBytes, _ := json.Marshal(createBuzzData)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/create", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	// Create fresh router for each call to avoid conflicts
	buzzRouter := gin.Default()
	buzzRouter.POST("/api/v1/buzz/create", middleware.Authorize(storage.Connection().Postgresql), ctrl.Create)
	buzzRouter.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated && resp.Code != http.StatusOK {
		t.Fatalf("Failed to create buzz, status: %d, body: %s", resp.Code, resp.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse create buzz response: %v", err)
	}

	data := result["data"].(map[string]interface{})
	return data
}

// Helper function to join buzz
func joinBuzz(t *testing.T, r *gin.Engine, ctrl buzzController.Controller, token, buzzID string) {
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/join", buzzID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	// Create fresh router for each call to avoid conflicts
	buzzRouter := gin.Default()
	buzzRouter.POST("/api/v1/buzz/:id/join", middleware.Authorize(storage.Connection().Postgresql), ctrl.Join)
	buzzRouter.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Failed to join buzz, status: %d, body: %s", resp.Code, resp.Body.String())
	}
}
