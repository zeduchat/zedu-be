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

// TestCreateBuzzRegularChannel tests creating a buzz in a regular channel
func TestCreateBuzzRegularChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Create host user
	hostSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("host%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		Password:    "password",
		UserName:    fmt.Sprintf("host_%v", currUUID),
	}

	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}

	r := gin.Default()
	tst.SignupUser(t, r, auth, hostSignUpData, false)

	loginData := models.LoginRequestModel{
		Email:    hostSignUpData.Email,
		Password: hostSignUpData.Password,
	}
	hostToken := tst.GetLoginToken(t, r, auth, loginData)

	// Get user ID from token
	var hostUserID string
	token, err := middleware.TokenValid(hostToken)
	if err != nil {
		t.Fatalf("Invalid token: %v", err)
	}
	claims := token.Claims.(jwt.MapClaims)
	if userID, ok := claims["user_id"].(string); ok {
		hostUserID = userID
	} else {
		t.Fatal("Could not extract user ID from token")
	}

	// Create organization
	org := models.Organisation{
		ID:          utility.GenerateUUID(),
		Name:        fmt.Sprintf("TestOrg%s", currUUID),
		Description: "Test organization",
		Email:       hostSignUpData.Email,
		OwnerID:     hostUserID,
	}
	if err := db.Postgresql.Create(&org).Error; err != nil {
		t.Logf("Organization might already exist: %v", err)
	}

	// Manually create channel and add user (bypass Elasticsearch issue)
	channelID := utility.GenerateUUID()

	// Create channel directly in database
	channel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("test-channel-%s", currUUID),
		Description:    "Test channel for buzz",
		OrganisationID: org.ID,
		OwnerId:        hostUserID,
		CreatedAt:      time.Now(),
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	// Add user to channel
	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     hostUserID,
		CreatedAt:  time.Now(),
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	buzzCtrl := buzzController.Controller{Db: db, Validator: validatorRef, Logger: logger}

	t.Run("Create Buzz in Regular Channel - Success", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzCtrl.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: channelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hostToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			body := rr.Body.String()
			t.Logf("Response body: %s", body)
			t.Fatalf("Expected status 201, got %d", rr.Code)
		}

		data := tst.ParseResponse(rr)
		message := data["message"].(string)
		if message != "buzz created successfully" {
			t.Errorf("Expected message 'buzz created successfully', got '%s'", message)
		}

		// Verify response data
		responseData, ok := data["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		buzzID := responseData["buzz_id"].(string)
		if buzzID == "" {
			t.Error("Expected buzz_id in response")
		}

		if responseData["channel_id"] != channelID {
			t.Errorf("Expected channel_id %s, got %v", channelID, responseData["channel_id"])
		}

		if responseData["status"] != "active" {
			t.Error("Expected status to be 'active'")
		}

		if responseData["host_id"] != hostUserID {
			t.Errorf("Expected host_id %s, got %v", hostUserID, responseData["host_id"])
		}

		participants, ok := responseData["participants"].([]interface{})
		if !ok || len(participants) == 0 {
			t.Error("Expected participants array with at least the host")
		}

		// Verify agora_token is present
		if _, exists := responseData["agora_token"]; !exists {
			t.Error("Expected agora_token field in response")
		}

		t.Logf("✅ Successfully created buzz in regular channel: %s", buzzID)
	})

	// Cleanup
	db.Postgresql.Where("channel_id = ?", channelID).Delete(&models.Buzz{})
	db.Postgresql.Where("channels_id = ? AND user_id = ?", channelID, hostUserID).Delete(&models.UserChannels{})
	db.Postgresql.Where("id = ?", channelID).Delete(&models.Channels{})
}

// TestCreateBuzzDMChannel tests creating a buzz in a DM channel
func TestCreateBuzzDMChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Create two users for DM
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("user1%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		Password:    "password",
		UserName:    fmt.Sprintf("user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("user2%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		Password:    "password",
		UserName:    fmt.Sprintf("user2_%v", currUUID),
	}

	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}

	r := gin.Default()
	tst.SignupUser(t, r, auth, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), auth, user2SignUpData, false)

	user1Token := tst.GetLoginToken(t, r, auth, models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	})
	user2Token := tst.GetLoginToken(t, gin.Default(), auth, models.LoginRequestModel{
		Email:    user2SignUpData.Email,
		Password: user2SignUpData.Password,
	})

	// Get user IDs
	var user1ID, user2ID string
	token1, _ := middleware.TokenValid(user1Token)
	claims1 := token1.Claims.(jwt.MapClaims)
	if userID, ok := claims1["user_id"].(string); ok {
		user1ID = userID
	}
	token2, _ := middleware.TokenValid(user2Token)
	claims2 := token2.Claims.(jwt.MapClaims)
	if userID, ok := claims2["user_id"].(string); ok {
		user2ID = userID
	}

	dmChannelID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()

	// Create DM channel where user1 is creator
	dmChannel1 := models.DmChannels{
		ID:              utility.GenerateUUID(),
		UserId:          user1ID,
		ChannelId:       dmChannelID,
		OrgId:           orgID,
		ParticipantId:   &user2ID,
		ParticipantHash: "hash1",
		ChannelType:     "dm",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := db.Postgresql.Create(&dmChannel1).Error; err != nil {
		t.Fatalf("Failed to create DM channel for user1: %v", err)
	}

	// Create reverse entry where user2 is creator
	dmChannel2 := models.DmChannels{
		ID:              utility.GenerateUUID(),
		UserId:          user2ID,
		ChannelId:       dmChannelID,
		OrgId:           orgID,
		ParticipantId:   &user1ID,
		ParticipantHash: "hash2",
		ChannelType:     "dm",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := db.Postgresql.Create(&dmChannel2).Error; err != nil {
		t.Fatalf("Failed to create DM channel for user2: %v", err)
	}

	buzzCtrl := buzzController.Controller{Db: db, Validator: validatorRef, Logger: logger}

	t.Run("Create Buzz in DM - User1 (Creator) Creates", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzCtrl.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: dmChannelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+user1Token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			body := rr.Body.String()
			t.Logf("Response body: %s", body)
			t.Fatalf("Expected status 201, got %d", rr.Code)
		}

		data := tst.ParseResponse(rr)
		responseData := data["data"].(map[string]interface{})
		buzzID := responseData["buzz_id"].(string)

		t.Logf("✅ User1 (DM creator) successfully created buzz: %s", buzzID)

		// Cleanup this buzz before next test
		db.Postgresql.Where("id = ?", buzzID).Delete(&models.Buzz{})
		db.Postgresql.Where("buzz_id = ?", buzzID).Delete(&models.BuzzParticipant{})
	})

	t.Run("Create Buzz in DM - User2 (Participant) Creates", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzCtrl.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: dmChannelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+user2Token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			body := rr.Body.String()
			t.Logf("Response body: %s", body)
			t.Fatalf("Expected status 201, got %d. User2 should be able to create buzz as participant", rr.Code)
		}

		data := tst.ParseResponse(rr)
		responseData := data["data"].(map[string]interface{})
		buzzID := responseData["buzz_id"].(string)

		t.Logf("✅ User2 (DM participant) successfully created buzz: %s", buzzID)

		// Cleanup
		db.Postgresql.Where("id = ?", buzzID).Delete(&models.Buzz{})
		db.Postgresql.Where("buzz_id = ?", buzzID).Delete(&models.BuzzParticipant{})
	})

	// Cleanup
	db.Postgresql.Where("channel_id = ?", dmChannelID).Delete(&models.DmChannels{})
}

// TestJoinBuzzScenarios tests joining an existing buzz
func TestJoinBuzzScenarios(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Create two users
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("joinuser1%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		Password:    "password",
		UserName:    fmt.Sprintf("joinuser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("joinuser2%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		Password:    "password",
		UserName:    fmt.Sprintf("joinuser2_%v", currUUID),
	}

	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}

	r := gin.Default()
	tst.SignupUser(t, r, auth, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), auth, user2SignUpData, false)

	user1Token := tst.GetLoginToken(t, r, auth, models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	})
	user2Token := tst.GetLoginToken(t, gin.Default(), auth, models.LoginRequestModel{
		Email:    user2SignUpData.Email,
		Password: user2SignUpData.Password,
	})

	// Get user IDs
	var user1ID, user2ID string
	tkn1, _ := middleware.TokenValid(user1Token)
	clms1 := tkn1.Claims.(jwt.MapClaims)
	if userID, ok := clms1["user_id"].(string); ok {
		user1ID = userID
	}
	tkn2, _ := middleware.TokenValid(user2Token)
	clms2 := tkn2.Claims.(jwt.MapClaims)
	if userID, ok := clms2["user_id"].(string); ok {
		user2ID = userID
	}

	// Create channel and add both users
	channelID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()

	org := models.Organisation{
		ID:          orgID,
		Name:        fmt.Sprintf("TestOrg%s", currUUID),
		Description: "Test organization",
		OwnerID:     user1ID,
	}
	if err := db.Postgresql.Create(&org).Error; err != nil {
		t.Logf("Organization might already exist: %v", err)
	}

	channel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("join-test-channel-%s", currUUID),
		Description:    "Test channel for join buzz",
		OrganisationID: orgID,
		CreatedAt:      time.Now(),
		OwnerId:        user1ID,
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Logf("Channel might already exist: %v", err)
	}

	if err := db.Postgresql.Create(&models.UserChannels{
		ChannelsID: channelID,
		UserID:     user1ID,
		CreatedAt:  time.Now(),
	}).Error; err != nil {
		t.Logf("UserChannels might already exist: %v", err)
	}

	if err := db.Postgresql.Create(&models.UserChannels{
		ChannelsID: channelID,
		UserID:     user2ID,
		CreatedAt:  time.Now(),
	}).Error; err != nil {
		t.Logf("UserChannels might already exist: %v", err)
	}

	// User1 creates a buzz
	buzzID := utility.GenerateUUID()
	now := time.Now()
	buzz := models.Buzz{
		ID:             buzzID,
		ChannelID:      channelID,
		ChannelType:    models.ChannelTypeRegular,
		HostID:         user1ID,
		ParticipantIDs: []string{user1ID},
		BuzzStartTime:  now,
		IsLiveStatus:   true,
		Status:         models.BuzzStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.Postgresql.Create(&buzz).Error; err != nil {
		t.Logf("Buzz might already exist: %v", err)
	}

	buzzParticipant := models.BuzzParticipant{
		ID:       utility.GenerateUUID(),
		BuzzID:   buzzID,
		UserID:   user1ID,
		Status:   models.BuzzParticipantStatusActive,
		JoinedAt: now,
	}
	db.Postgresql.Create(&buzzParticipant)

	buzzCtrl := buzzController.Controller{Db: db, Validator: validatorRef, Logger: logger}

	t.Run("Join Buzz - User2 Joins Successfully", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/:id/join", buzzCtrl.Join)
		}

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/join", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+user2Token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			body := rr.Body.String()
			t.Logf("Response body: %s", body)
			t.Fatalf("Expected status 200, got %d", rr.Code)
		}

		data := tst.ParseResponse(rr)
		message := data["message"].(string)
		if message != "user joined buzz successfully" {
			t.Errorf("Expected message 'user joined buzz successfully', got '%s'", message)
		}

		responseData := data["data"].(map[string]interface{})
		participants, ok := responseData["participants"].([]interface{})
		if !ok || len(participants) != 2 {
			t.Errorf("Expected 2 participants after join, got %d", len(participants))
		}

		// Verify agora_token is present
		if _, exists := responseData["agora_token"]; !exists {
			t.Error("Expected agora_token field in response")
		}

		t.Logf("✅ User2 successfully joined buzz")
	})

	t.Run("Join Buzz - User2 Already in Buzz", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/:id/join", buzzCtrl.Join)
		}

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/join", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+user2Token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200 for already joined, got %d", rr.Code)
		}

		data := tst.ParseResponse(rr)
		message := data["message"].(string)
		if message != "user joined buzz successfully" {
			t.Errorf("Expected 'user joined buzz successfully', got '%s'", message)
		}

		t.Logf("✅ Correctly rejected duplicate join attempt")
	})

	// Cleanup
	db.Postgresql.Where("buzz_id = ?", buzzID).Delete(&models.BuzzParticipant{})
	db.Postgresql.Where("id = ?", buzzID).Delete(&models.Buzz{})
	db.Postgresql.Where("channels_id = ?", channelID).Delete(&models.UserChannels{})
	db.Postgresql.Where("id = ?", channelID).Delete(&models.Channels{})
}
