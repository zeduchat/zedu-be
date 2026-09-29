package test_channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestChannelsEndpoints(t *testing.T) {
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

	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}
	channelController := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()
	tst.SignupUser(t, r, auth, userSignUpData, false)

	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	token := tst.GetLoginToken(t, r, auth, loginData)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam%s", currUUID),
		Description: "Some Random description",
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, org, createOrgData, token)

	createChannelsData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("Mr%sChannels", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Some Random description",
	}

	channels_id, channelName := tst.CreateChannels(t, r, channelController, db, createChannelsData, token)

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test2",
		LastName:    "user2",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username2_%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), auth, user2SignUpData, false)
	token2 := tst.GetLoginToken(t, gin.Default(), auth, models.LoginRequestModel{Email: user2SignUpData.Email, Password: user2SignUpData.Password})

	var user2Model models.User
	db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2Model)
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user2Model.ID,
		OrganisationID: orgId,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	tests := []struct {
		Name         string
		RequestBody  any
		ExpectedCode int
		Message      string
		Method       string
		Headers      map[string]string
		RequestURI   url.URL
	}{
		{
			Name: "Create Channels Action",
			RequestBody: models.CreateChannelsRequest{
				Name:           "Test-Channels",
				Description:    "This is a test channel",
				Username:       userSignUpData.UserName,
				OrganisationID: orgId,
			},
			ExpectedCode: http.StatusCreated,
			Message:      "Channel Created Successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: "/api/v1/channels/"},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Get Channels Action",
			ExpectedCode: http.StatusOK,
			Message:      "channel retreived successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/channels/%s", channels_id)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},

		{
			Name:         "Update Channels Username Action",
			ExpectedCode: http.StatusOK,
			Message:      "username updated successfully",
			RequestBody: models.UpdateChannelsUserNameReq{
				Username: fmt.Sprintf("username%v", currUUID),
			},
			Method:     http.MethodPatch,
			RequestURI: url.URL{Path: fmt.Sprintf("/api/v1/channels/%s/username", channels_id)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		}, {
			Name:         "Update Channels Action",
			ExpectedCode: http.StatusOK,
			RequestBody: models.UpdateChannelsRequest{
				Name: "Normal",
			},
			Message:    "Channels updated successfully",
			Method:     http.MethodPatch,
			RequestURI: url.URL{Path: fmt.Sprintf("/api/v1/channels/%s", channels_id)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Check User In Channels Action",
			ExpectedCode: http.StatusOK,
			RequestBody: models.UpdateChannelsRequest{
				Name: "Normal",
			},
			Message:    "user checked successfully",
			Method:     http.MethodGet,
			RequestURI: url.URL{Path: fmt.Sprintf("/api/v1/channels/%s/user-exist", channels_id)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		}, {
			Name:         "Get Channels by Name Action",
			ExpectedCode: http.StatusOK,
			RequestBody: models.UpdateChannelsRequest{
				Name: "Normal",
			},
			Message:    "channel name retrieved successfully",
			Method:     http.MethodGet,
			RequestURI: url.URL{Path: fmt.Sprintf("/api/v1/channels/name/%s", "Test-Channels")},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		}, {
			Name:         "Search Channels by Name Action",
			Message:      "channel names retrieved successfully",
			ExpectedCode: http.StatusOK,
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/channels/search/%s", channelName)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Join Channels Action",
			ExpectedCode: http.StatusOK,
			Message:      "channel joined successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/channels/%s/join", channels_id)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token2,
			},
		},
		{
			Name:         "Leave Channels Action",
			ExpectedCode: http.StatusOK,
			Message:      "user left channel successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/channels/%s/leave", channels_id)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Delete Channels Action",
			ExpectedCode: http.StatusOK,
			Message:      "channel deleted successfully",
			Method:       http.MethodDelete,
			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/channels/%s", channels_id)},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	channel := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}

	for _, test := range tests {
		r := gin.Default()

		channelUrl := r.Group(fmt.Sprintf("%v", "/api/v1/channels"), middleware.Authorize(db.Postgresql))
		{
			channelUrl.POST("/", channel.CreateChannel)
			channelUrl.GET("/:channelId", channel.GetChannel)
			channelUrl.POST("/:channelId/join", channel.JoinChannels)
			channelUrl.POST("/:channelId/leave", channel.LeaveChannels)
			channelUrl.PATCH("/:channelId/username", channel.UpdateUsername)
			channelUrl.GET("/name/:channelName", channel.GetChannelsByName)
			channelUrl.GET("/:channelId/num-users", channel.CountChannelsUsers)
			channelUrl.PATCH("/:channelId", channel.UpdateChannels)
			channelUrl.DELETE("/:channelId", channel.DeleteChannel)
			channelUrl.GET("/:channelId/user-exist", channel.CheckUser)
			channelUrl.GET(("/search/:channelName"), channel.SearchChannelsByNames)
		}

		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			req, err := http.NewRequest(test.Method, test.RequestURI.String(), &b)
			if err != nil {
				t.Fatal(err)
			}

			for i, v := range test.Headers {
				req.Header.Set(i, v)
			}

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

			if test.Name == "Get Channels Action" {
				responseData, ok := data["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected data object in Get Channels Action response")
				}
				if _, exists := responseData["users"]; !exists {
					t.Errorf("expected users field in response")
				}
				if _, exists := responseData["total_user_count"]; !exists {
					t.Errorf("expected total_user_count field in response")
				}
				if _, exists := responseData["is_restricted"]; !exists {
					t.Errorf("expected is_restricted field in response")
				}
			}
		})

	}

}

func TestToggleUserJoinedMessage(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("toggleuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("toggle_username%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelController := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authCtrl, loginData)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("ToggleOrg%s", currUUID),
		Description: "Org for toggle user joined message test",
		Email:       userSignUpData.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token)

	createChannelsData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("ToggleChan%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("togglechan%s", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Channel for toggle user joined message test",
	}

	channelID, _ := tst.CreateChannels(t, r, channelController, db, createChannelsData, token)

	defer func() {
		_ = db.Postgresql.Where("id = ?", channelID).Delete(&models.Channels{}).Error
		_ = db.Postgresql.Where("id = ?", orgId).Delete(&models.Organisation{}).Error
		_ = db.Postgresql.Where("email = ?", userSignUpData.Email).Delete(&models.User{}).Error
	}()

	rGroup := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	rGroup.PUT("/:channelId/toggle-user-joined-message", channelController.ToggleUserJoinedMessage)
	rGroup.POST("/:channelId/join", channelController.JoinChannels)

	t.Run("Successfully Toggle User Joined Message to False and Verify No Join System Message Created", func(t *testing.T) {
		show := false
		body := models.ToggleJoinedMessageRequest{
			ShowJoinedMessage: &show,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(body)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%s/toggle-user-joined-message", channelID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		tst.AssertResponseMessage(t, data["message"].(string), "channel user joined message setting updated successfully")

		// Create user 2 and join channel
		currUUID2 := utility.GenerateUUID()
		user2SignUp := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("toggleuser2_%v@qa.team", currUUID2),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "test2",
			LastName:    "user2",
			Password:    "password",
			UserName:    fmt.Sprintf("toggle_user2_%v", currUUID2),
		}
		tst.SignupUser(t, r, authCtrl, user2SignUp, false)
		token2 := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: user2SignUp.Email, Password: user2SignUp.Password})

		defer func() {
			_ = db.Postgresql.Where("email = ?", user2SignUp.Email).Delete(&models.User{}).Error
		}()

		joinReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%s/join", channelID), strings.NewReader("{}"))
		joinReq.Header.Set("Content-Type", "application/json")
		joinReq.Header.Set("Authorization", "Bearer "+token2)

		rrJoin := httptest.NewRecorder()
		r.ServeHTTP(rrJoin, joinReq)

		tst.AssertStatusCode(t, rrJoin.Code, http.StatusOK)

		// Verify no join system message exists for channel
		var count int64
		_ = db.Postgresql.Model(&models.MessageDocument{}).Where("channels_id = ? AND content LIKE ?", channelID, "%joined this channel%").Count(&count).Error
		if count != 0 {
			t.Errorf("expected 0 join system messages when show_joined_message is false, got %d", count)
		}
	})
}
