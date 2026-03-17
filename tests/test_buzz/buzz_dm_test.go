package test_buzz

import (
	"bytes"
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
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestBuzzCreateInDM tests creating a buzz in a DM channel
func TestBuzzCreateInDM(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Create two users for DM
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

	// Setup users
	r := gin.Default()
	tst.SignupUser(t, r, authController, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user2SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

	// Get user IDs from database
	var user1, user2 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error; err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}

	// Create organization for DM context
	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	// Create a DM channel
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

	// Create reverse DM channel entry for user2
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

	buzzController := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}

	t.Run("Create Buzz in DM - Success", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: dmChannelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rr.Code)
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify response contains data
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response missing data field")
		}

		// Verify buzz was created with correct channel_type
		buzzID, ok := data["buzz_id"].(string)
		if !ok {
			t.Fatal("Response missing buzz_id")
		}

		// Fetch buzz from database and verify channel_type
		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz from database: %v", err)
		}

		if buzz.ChannelType != models.ChannelTypeDM {
			t.Errorf("Expected channel_type to be '%s', got '%s'", models.ChannelTypeDM, buzz.ChannelType)
		}

		if buzz.ChannelID != dmChannelID {
			t.Errorf("Expected channel_id to be '%s', got '%s'", dmChannelID, buzz.ChannelID)
		}

		t.Logf("✅ Buzz created successfully in DM with channel_type: %s", buzz.ChannelType)
	})

	t.Run("Create Buzz in DM - User Not in Channel", func(t *testing.T) {
		// Create a third user who is not part of the DM
		user3SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("dmuser3_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "DMUser",
			LastName:    "Three",
			Password:    "password",
			UserName:    fmt.Sprintf("dmuser3_%v", currUUID),
		}
		loginData3 := models.LoginRequestModel{
			Email:    user3SignUpData.Email,
			Password: user3SignUpData.Password,
		}

		tst.SignupUser(t, gin.Default(), authController, user3SignUpData, false)
		token3 := tst.GetLoginToken(t, gin.Default(), authController, loginData3)

		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: dmChannelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token3)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		t.Logf("Response Status: %d", rr.Code)
		t.Logf("Response Body: %s", rr.Body.String())

		if rr.Code != http.StatusForbidden && rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 403 or 400, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		message, _ := response["message"].(string)
		t.Logf("✅ Correctly rejected non-member with message: %s", message)
	})

	t.Run("Create Buzz in DM - Already Active Buzz", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: dmChannelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		code := int(data["status_code"].(float64))
		tst.AssertStatusCode(t, code, http.StatusOK)

		message := data["message"].(string)
		tst.AssertResponseMessage(t, message, "buzz joined successfully")

		t.Logf("✅ Successfully returned existing buzz in DM")
	})
}

// TestBuzzCreateInGroupDM tests creating a buzz in a group DM channel
func TestBuzzCreateInGroupDM(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Create three users for group DM
	users := make([]models.User, 3)
	tokens := make([]string, 3)

	for i := 0; i < 3; i++ {
		signUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("groupuser%d_%v@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   fmt.Sprintf("GroupUser%d", i),
			LastName:    "Test",
			Password:    "password",
			UserName:    fmt.Sprintf("groupuser%d_%v", i, currUUID),
		}
		loginData := models.LoginRequestModel{
			Email:    signUpData.Email,
			Password: signUpData.Password,
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
		tst.SignupUser(t, r, authController, signUpData, false)
		tokens[i] = tst.GetLoginToken(t, r, authController, loginData)

		if err := db.Postgresql.Where("email = ?", signUpData.Email).First(&users[i]).Error; err != nil {
			t.Fatalf("Failed to get user%d: %v", i, err)
		}
	}

	// Get organization
	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", users[0].ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	// Create a group DM channel
	groupDMChannelID := utility.GenerateUUID()

	// Create channel_participants entries for all users
	for i := 0; i < 3; i++ {
		participant := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDMChannelID,
			UserId:    users[i].ID,
			OrgId:     org.ID,
		}
		if err := db.Postgresql.Create(&participant).Error; err != nil {
			t.Fatalf("Failed to create channel participant %d: %v", i, err)
		}
	}

	// Create group DM channel entries
	participantHash := utility.GenerateUUID() // Simplified hash for testing

	for i := 0; i < 3; i++ {
		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          users[i].ID,
			ChannelId:       groupDMChannelID,
			OrgId:           org.ID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}

		if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
			t.Fatalf("Failed to create group DM channel for user %d: %v", i, err)
		}
	}

	buzzController := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}

	t.Run("Create Buzz in Group DM - Success", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: groupDMChannelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokens[0])

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		t.Logf("Response Status: %d", rr.Code)
		t.Logf("Response Body: %s", rr.Body.String())

		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rr.Code)
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response missing data field")
		}

		buzzID, ok := data["buzz_id"].(string)
		if !ok {
			t.Fatal("Response missing buzz_id")
		}

		// Verify buzz in database
		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz from database: %v", err)
		}

		if buzz.ChannelType != models.ChannelTypeGroupDM {
			t.Errorf("Expected channel_type to be '%s', got '%s'", models.ChannelTypeGroupDM, buzz.ChannelType)
		}

		if buzz.ChannelID != groupDMChannelID {
			t.Errorf("Expected channel_id to be '%s', got '%s'", groupDMChannelID, buzz.ChannelID)
		}

		t.Logf("✅ Buzz created successfully in Group DM with channel_type: %s", buzz.ChannelType)
	})

	// Cleanup: End the buzz for next test
	db.Postgresql.Exec("UPDATE buzzs SET status = 'ended', is_live_status = false WHERE channel_id = ?", groupDMChannelID)

	t.Run("Any Group Member Can Create Buzz", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		// User 2 (not the creator of group) creates a buzz
		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: groupDMChannelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokens[1])

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		t.Logf("Response Status: %d", rr.Code)
		t.Logf("Response Body: %s", rr.Body.String())

		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rr.Code)
		}

		t.Logf("✅ Any group member can successfully create a buzz")
	})
}
