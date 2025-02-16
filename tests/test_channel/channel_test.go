package test_channel

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

	tests := []struct {
		Name         string
		RequestBody  interface{}
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
			channelUrl.POST("/", channel.CreateChannels)
			channelUrl.GET("/:channelId", channel.GetChannels)
			channelUrl.POST("/:channelId/join", channel.JoinChannels)
			channelUrl.POST("/:channelId/leave", channel.LeaveChannels)
			channelUrl.PATCH("/:channelId/username", channel.UpdateUsername)
			channelUrl.GET("/name/:channelName", channel.GetChannelsByName)
			channelUrl.GET("/:channelId/num-users", channel.CountChannelsUsers)
			channelUrl.PATCH("/:channelId", channel.UpdateChannels)
			channelUrl.DELETE("/:channelId", channel.DeleteChannels)
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
		})

	}

}
