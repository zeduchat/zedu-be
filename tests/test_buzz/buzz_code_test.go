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

func TestBuzzCodeBackwardCompatibility(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

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

	user2UUID := utility.GenerateUUID()
	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser2%v@qa.team", user2UUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test2",
		LastName:    "user2",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username2%v", user2UUID),
	}
	login2Data := models.LoginRequestModel{
		Email:    user2SignUpData.Email,
		Password: user2SignUpData.Password,
	}

	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}

	r := gin.Default()
	tst.SignupUser(t, r, auth, userSignUpData, false)
	token := tst.GetLoginToken(t, r, auth, loginData)

	r2 := gin.Default()
	tst.SignupUser(t, r2, auth, user2SignUpData, false)
	token2 := tst.GetLoginToken(t, r2, auth, login2Data)

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

	channelID := utility.GenerateUUID()

	channel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
		Description:    "Some Random description",
		OrganisationID: orgId,
		OwnerId:        getTestUserID(db, userSignUpData.Email),
		CreatedAt:      time.Now(),
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	hostUserID := getTestUserID(db, userSignUpData.Email)
	hostUserChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     hostUserID,
		Username:   userSignUpData.UserName,
		CreatedAt:  time.Now(),
	}
	if err := db.Postgresql.Create(&hostUserChannel).Error; err != nil {
		t.Logf("Warning: Failed to add host to channel: %v", err)
	}

	var userMgmt models.OrgUserManagement
	var orgRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", "Administrator").First(&orgRole).Error; err != nil {
		t.Fatalf("Failed to get org role: %v", err)
	}
	userMgmt.OrganisationID = orgId
	userMgmt.UserID = getTestUserID(db, user2SignUpData.Email)
	userMgmt.RoleID = orgRole.ID
	userMgmt.Status = "active"
	db.Postgresql.Create(&userMgmt)

	var userChannel models.UserChannels
	userChannel.ChannelsID = channelID
	userChannel.UserID = userMgmt.UserID
	userChannel.Username = user2SignUpData.UserName
	db.Postgresql.Create(&userChannel)

	buzzController := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	createBuzzReq := models.CreateBuzzRequest{
		ChannelID: channelID,
	}

	var buzzID string
	var buzzCode string
	{
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/create", buzzController.Create)
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		data := tst.ParseResponse(rr)
		if responseData, ok := data["data"].(map[string]interface{}); ok {
			buzzID = responseData["buzz_id"].(string)
			buzzCode = responseData["buzz_code"].(string)

			if buzzCode == "" {
				t.Fatal("Expected buzz_code in create response")
			}

			expectedCode := utility.ExtractBuzzCode(buzzID)
			if buzzCode != expectedCode {
				t.Errorf("Expected buzz_code %s, got %s", expectedCode, buzzCode)
			}
		}
	}

	t.Run("Join Buzz with BuzzCode", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/:id/join", buzzController.Join)
		}

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/join", buzzCode), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		code := int(data["status_code"].(float64))
		tst.AssertStatusCode(t, code, http.StatusOK)

		responseData, ok := data["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		if responseData["buzz_code"] != buzzCode {
			t.Errorf("Expected buzz_code %s in response, got %v", buzzCode, responseData["buzz_code"])
		}
	})

	t.Run("Get Metadata with Full UUID (Backward Compatibility)", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.GET("/:id/metadata", buzzController.GetMetadata)
		}

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/metadata", buzzID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		code := int(data["status_code"].(float64))
		tst.AssertStatusCode(t, code, http.StatusOK)

		responseData, ok := data["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		if responseData["buzz_id"] != buzzID {
			t.Errorf("Expected buzz_id %s in response, got %v", buzzID, responseData["buzz_id"])
		}

		if responseData["buzz_code"] != buzzCode {
			t.Errorf("Expected buzz_code %s in response, got %v", buzzCode, responseData["buzz_code"])
		}
	})

	t.Run("Get Metadata with BuzzCode", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.GET("/:id/metadata", buzzController.GetMetadata)
		}

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/metadata", buzzCode), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		responseData, ok := data["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data field in response")
		}

		if responseData["buzz_code"] != buzzCode {
			t.Errorf("Expected buzz_code %s in response, got %v", buzzCode, responseData["buzz_code"])
		}
	})

	t.Run("Invalid BuzzCode Format", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/:id/join", buzzController.Join)
		}

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/INVALIDCODE/join", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

		data := tst.ParseResponse(rr)
		message := data["message"].(string)
		if message != "invalid buzz code or ID format" {
			t.Errorf("Expected error message 'invalid buzz code or ID format', got '%s'", message)
		}
	})

	t.Run("Non-existent BuzzCode", func(t *testing.T) {
		r := gin.Default()
		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/:id/join", buzzController.Join)
		}

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/AABBCCDDEEFF/join", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)

		data := tst.ParseResponse(rr)
		message := data["message"].(string)
		if message != "buzz not found" {
			t.Errorf("Expected error message 'buzz not found', got '%s'", message)
		}
	})
}
