package test_huddle

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
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/huddle"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestHuddleEndpoints(t *testing.T) {
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

	// Create second user (will join huddle)
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

	// Create huddle
	huddleController := huddle.Controller{Db: db, Validator: validatorRef, Logger: logger}
	createHuddleReq := models.CreateHuddleRequest{
		ChannelID: channelID,
	}

	var huddleID string
	{
		r := gin.Default()
		huddleUrl := r.Group("/api/v1/huddles", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			huddleUrl.POST("/create", huddleController.Create)
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createHuddleReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/huddles/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		data := tst.ParseResponse(rr)
		if responseData, ok := data["data"].(map[string]interface{}); ok {
			huddleID = responseData["huddle_id"].(string)
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
			Name: "Create Huddle Action",
			RequestBody: models.CreateHuddleRequest{
				ChannelID: channelID,
			},
			ExpectedCode: http.StatusCreated,
			Message:      "huddle created successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: "/api/v1/huddles/create"},
			Token:        token,
		},
		{
			Name:         "Join Huddle Action - Success",
			RequestBody:  nil,
			ExpectedCode: http.StatusOK,
			Message:      "user joined huddle successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/huddles/%s/join", huddleID)},
			Token:        token2,
		},
		{
			Name:         "Join Huddle Action - Already Joined",
			RequestBody:  nil,
			ExpectedCode: http.StatusBadRequest,
			Message:      "user is already in the huddle",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/huddles/%s/join", huddleID)},
			Token:        token,
		},
		{
			Name:         "Join Huddle Action - Invalid Huddle ID",
			RequestBody:  nil,
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid huddle ID format",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: "/api/v1/huddles/invalid-uuid/join"},
			Token:        token,
		},
		{
			Name:         "Join Huddle Action - Non-existent Huddle",
			RequestBody:  nil,
			ExpectedCode: http.StatusBadRequest,
			Message:      "huddle does not exist",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/huddles/%s/join", utility.GenerateUUID())},
			Token:        token,
		},
	}

	for _, test := range tests {
		r := gin.Default()

		huddleUrl := r.Group("/api/v1/huddles", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
		{
			huddleUrl.POST("/create", huddleController.Create)
			huddleUrl.POST("/:id/join", huddleController.Join)
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
