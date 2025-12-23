package test_buzz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuzzJoin(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Create first user (host)
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

	// Create second user (will join buzz)
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
	channelController := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}

	// Setup first user
	r := gin.Default()
	tst.SignupUser(t, r, auth, userSignUpData, false)
	token := tst.GetLoginToken(t, r, auth, loginData)

	// Setup second user with new router
	r2 := gin.Default()
	tst.SignupUser(t, r2, auth, user2SignUpData, false)
	token2 := tst.GetLoginToken(t, r2, auth, login2Data)

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

	// Create channel
	createChannelsData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("Mr%sChannels", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Some Random description",
	}
	channelID, _ := tst.CreateChannels(t, r, channelController, db, createChannelsData, token)

	// Add second user to organization
	var userMgmt models.OrgUserManagement
	var orgRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", "Administrator").First(&orgRole).Error; err != nil {
		t.Fatalf("Failed to get org role: %v", err)
	}
	userMgmt.OrganisationID = orgId
	userMgmt.UserID = getUserID(t, db, user2SignUpData.Email)
	userMgmt.RoleID = orgRole.ID
	userMgmt.Status = "active"
	db.Postgresql.Create(&userMgmt)

	// Add second user to channel
	var userChannel models.UserChannels
	userChannel.ChannelsID = channelID
	userChannel.UserID = userMgmt.UserID
	userChannel.Username = user2SignUpData.UserName
	db.Postgresql.Create(&userChannel)

	// Create buzz
	buzzController := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	createBuzzReq := models.CreateBuzzRequest{
		ChannelID: channelID,
	}

	var buzzID string
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
		}
	}

	tests := []struct {
		Name         string
		RequestBody  any
		ExpectedCode int
		Message      string
		Method       string
		Headers      map[string]string
		RequestURI   url.URL
		Token        string
	}{
		{
			Name:         "Join Buzz Action - Success",
			RequestBody:  nil,
			ExpectedCode: http.StatusOK,
			Message:      "user joined buzz successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/buzz/%s/join", buzzID)},
			Token:        token2,
		},
		{
			Name:         "Join Buzz Action - Already Joined",
			RequestBody:  nil,
			ExpectedCode: http.StatusBadRequest,
			Message:      "user is already in the buzz",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/buzz/%s/join", buzzID)},
			Token:        token,
		},
		{
			Name:         "Join Buzz Action - Invalid Buzz ID",
			RequestBody:  nil,
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid buzz ID format",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: "/api/v1/buzz/invalid-uuid/join"},
			Token:        token,
		},
		{
			Name:         "Join Buzz Action - Non-existent Buzz",
			RequestBody:  nil,
			ExpectedCode: http.StatusNotFound,
			Message:      "buzz does not exist",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/buzz/%s/join", utility.GenerateUUID())},
			Token:        token,
		},
	}

	for _, test := range tests {
		r := gin.Default()

		buzzUrl := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			buzzUrl.POST("/:id/join", buzzController.Join)
		}

		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			if test.RequestBody != nil {
				json.NewEncoder(&b).Encode(test.RequestBody)
			}

			req, err := http.NewRequest(test.Method, test.RequestURI.String(), &b)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+test.Token)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)

			data := tst.ParseResponse(rr)

			code := int(data["status_code"].(float64))
			tst.AssertStatusCode(t, code, test.ExpectedCode)

			if test.Message != "" {
				message := data["message"]
				if message != nil {
					tst.AssertResponseMessage(t, message.(string), test.Message)
				} else {
					tst.AssertResponseMessage(t, "", test.Message)
				}
			}

			// Additional assertions for successful join
			if test.Name == "Join Buzz Action - Success" && test.ExpectedCode == http.StatusOK {
				responseData, ok := data["data"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected data field in response")
				}

				// Verify agora_token is present in response (may be nil if Agora not configured)
				if agoraToken, ok := responseData["agora_token"].(map[string]interface{}); ok && agoraToken != nil {
					// Verify agora_token has required fields when present
					if agoraToken["token"] == nil || agoraToken["token"].(string) == "" {
						t.Error("Expected non-empty token in agora_token")
					}
					if agoraToken["app_id"] == nil || agoraToken["app_id"].(string) == "" {
						t.Error("Expected non-empty app_id in agora_token")
					}
					if agoraToken["channel_name"] == nil || agoraToken["channel_name"].(string) == "" {
						t.Error("Expected non-empty channel_name in agora_token")
					}
					if agoraToken["uid"] == nil {
						t.Error("Expected uid field in agora_token")
					}
				}
				// Note: agora_token can be nil if Agora service is not configured (e.g., in tests)

				// Verify other join response fields
				if responseData["Buzz_id"] != buzzID {
					t.Errorf("Expected buzz_id %s, got %v", buzzID, responseData["Buzz_id"])
				}
				if responseData["channel_id"] != channelID {
					t.Errorf("Expected channel_id %s, got %v", channelID, responseData["channel_id"])
				}
				   // Check for 'participants' field (should be a non-empty array)
				   participants, ok := responseData["participants"].([]interface{})
				   if !ok || len(participants) == 0 {
					   t.Error("Expected non-empty participants array in response")
				   }
			}
		})
	}
}

// Helper function to get user ID by email
func getUserID(t *testing.T, db *storage.Database, email string) string {
	var user models.User
	if err := db.Postgresql.Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}
	return user.ID
}
