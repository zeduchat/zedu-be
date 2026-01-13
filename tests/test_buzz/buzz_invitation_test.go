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

func TestBuzzInvitations(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	db := storage.Connection()
	validatorRef := validator.New()

	// --- SETUP: Create 2 users, organization, channel, buzz ---

	// U1 = inviter
	currUUID := utility.GenerateUUID()
	user1Signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("inviter%s@qa.team", currUUID),
		Password:    "password",
		UserName:    "inviter_user" + currUUID,
		FirstName:   "Inviter",
		LastName:    "User",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
	}
	user1Login := models.LoginRequestModel{Email: user1Signup.Email, Password: user1Signup.Password}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, user1Signup, false)
	token1 := tst.GetLoginToken(t, r, authCtrl, user1Login)

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgReq := models.CreateOrgRequestModel{
		Name:        "InvitationOrg" + currUUID,
		Description: "desc",
		Email:       user1Signup.Email,
		Type:        "type",
		Location:    "loc",
		Country:     "kenya",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, orgReq, token1)

	// Get user1 ID for channel owner
	var user1 models.User
	if err := db.Postgresql.Where("email = ?", user1Signup.Email).First(&user1).Error; err != nil {
		t.Fatalf("failed to fetch user1: %v", err)
	}

	// Manually create channel and add user1 (bypass Elasticsearch issue)
	channelID := utility.GenerateUUID()

	// Create channel directly in database
	channel := models.Channels{
		ID:             channelID,
		Name:           "TestChannel" + currUUID,
		OrganisationID: orgID,
		Description:    "desc",
		OwnerId:        user1.ID,
		CreatedAt:      time.Now(),
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	// Add user1 to channel
	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user1.ID,
		Username:   user1Signup.UserName,
		CreatedAt:  time.Now(),
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Logf("Warning: Failed to add user1 to channel: %v", err)
	}

	// create buzz
	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r2 := gin.Default()
	buzzGroup := r2.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql))
	buzzGroup.POST("/create", buzzCtrl.Create)

	createBuzzReq := models.CreateBuzzRequest{ChannelID: channelID}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(createBuzzReq)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r2.ServeHTTP(rr, req)

	data := tst.ParseResponse(rr)
	buzzID := data["data"].(map[string]interface{})["buzz_id"].(string)

	// U2 = invitee
	currUUID2 := utility.GenerateUUID()
	user2Signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("invitee%s@qa.team", currUUID2),
		Password:    "password",
		UserName:    "invitee_user" + currUUID2,
		FirstName:   "Invitee",
		LastName:    "User",
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
	}
	user2Login := models.LoginRequestModel{Email: user2Signup.Email, Password: user2Signup.Password}

	r = gin.Default()
	tst.SignupUser(t, r, authCtrl, user2Signup, false)
	token2 := tst.GetLoginToken(t, r, authCtrl, user2Login)

	// Get user2 ID
	var user2 models.User
	if err := db.Postgresql.Where("email = ?", user2Signup.Email).First(&user2).Error; err != nil {
		t.Fatalf("failed to fetch user2: %v", err)
	}

	// Add user2 to channel manually
	user2Channel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user2.ID,
		Username:   user2Signup.UserName,
		CreatedAt:  time.Now(),
	}
	if err := db.Postgresql.Create(&user2Channel).Error; err != nil {
		t.Fatal(err)
	}

	// --- TEST ROUTER FOR INVITATIONS ---
	router, _ := SetupBuzzInvitationTestRouter(logger, validatorRef)

	t.Run("SearchChannelMembers", func(t *testing.T) {
		body := models.SearchChannelMembersRequest{
			ChannelID: channelID,
			BuzzID:    buzzID,
			Query:     "invitee",
		}
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(body)

		req, _ := http.NewRequest("POST", "/api/v1/buzz/search-members", &buf)
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
	})

	t.Run("InviteUsersToBuzz - Success", func(t *testing.T) {
		body := models.InviteUsersToBuzzRequest{
			BuzzID:     buzzID,
			InviteeIDs: []string{getTestUserID(db, user2Signup.Email)},
		}
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(body)

		req, _ := http.NewRequest("POST", "/api/v1/buzz/invite", &buf)
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)
	})

	var invitationID string
	t.Run("GetPendingInvitations", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/buzz/invitations/pending", nil)
		req.Header.Set("Authorization", "Bearer "+token2)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		invs := data["data"].(map[string]interface{})["invitations"].([]interface{})
		invite := invs[0].(map[string]interface{})

		invitationID = invite["invitation_id"].(string)
	})

	t.Run("RespondToInvitation - Accept", func(t *testing.T) {
		body := models.RespondToInvitationRequest{
			InvitationID: invitationID,
			Accept:       true,
		}
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(body)

		req, _ := http.NewRequest("POST", "/api/v1/buzz/invitation/respond", &buf)
		req.Header.Set("Authorization", "Bearer "+token2)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
	})
}
