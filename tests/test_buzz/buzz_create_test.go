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

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuzzCreate(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Create user (host)
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}

	// Setup user
	r := gin.Default()
	tst.SignupUser(t, r, auth, userSignUpData, false)
	token := tst.GetLoginToken(t, r, auth, loginData)

	// Create organization
	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam%s", currUUID),
		Description: "Some Random description",
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgId, _, _ := tst.CreateOrganisation(t, r, db, org, createOrgData, token)

	// Get user ID for channel owner
	var user models.User
	if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}

	// Manually create channel and add user (bypass Elasticsearch issue)
	channelID := utility.GenerateUUID()

	// Create channel directly in database
	channel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
		Description:    "Some Random description",
		OrganisationID: orgId,
		OwnerId:        user.ID,
		CreatedAt:      time.Now(),
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	// Add user to channel
	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user.ID,
		Username:   userSignUpData.UserName,
		CreatedAt:  time.Now(),
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Logf("Warning: Failed to add user to channel: %v", err)
	}

	buzzController := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}

	t.Run("Create Buzz - Success", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: channelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		code := int(data["status_code"].(float64))
		tst.AssertStatusCode(t, code, http.StatusCreated)

		message := data["message"].(string)
		tst.AssertResponseMessage(t, message, "buzz created successfully")

		// Verify response data
		responseData, ok := data["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		if responseData["buzz_id"] == nil {
			t.Error("Expected buzz_id in response")
		}
		if responseData["channel_id"] != channelID {
			t.Errorf("Expected channel_id %s, got %v", channelID, responseData["channel_id"])
		}
		if responseData["status"] != "active" {
			t.Error("Expected status to be 'active'")
		}

		if responseData["ended_at"] == nil {
			t.Error("Expected ended_at in response")
		}

		// Check for 'participants' field (should be a non-empty array)
		participants, ok := responseData["participants"].([]interface{})
		if !ok || len(participants) == 0 {
			t.Error("Expected non-empty participants array in response")
		}
		// Verify agora_token is present (can be nil if generation fails)
		if _, exists := responseData["agora_token"]; !exists {
			t.Error("Expected agora_token field in response")
		}
	})

	t.Run("Create Buzz - Duplicate (Channel Already Has Active Buzz)", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		createBuzzReq := models.CreateBuzzRequest{
			ChannelID: channelID,
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		code := int(data["status_code"].(float64))
		tst.AssertStatusCode(t, code, http.StatusCreated)

		message := data["message"].(string)
		tst.AssertResponseMessage(t, message, "buzz created successfully") // JoinBuzz returns this message
	})
}
