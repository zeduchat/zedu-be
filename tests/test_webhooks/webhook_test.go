package test_webhook

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
	"github.com/hngprojects/telex_be/pkg/controller/webhook"
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

	channels_id, _:= tst.CreateChannels(t, r, channelController, db, createChannelsData, token)
	webhook_path := fmt.Sprintf("/api/v1/webhooks/%s", channels_id)

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
			Name:         "Create Webhook Action",
			ExpectedCode: http.StatusCreated,
			Message:      "webhook created successfully",
			Method:       http.MethodPost,
			RequestURI:   url.URL{Path: webhook_path},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Get Webhook Action",
			ExpectedCode: http.StatusOK,
			Message:      "webhook fetched successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: webhook_path},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		}, {
			Name:         "Get All Webhook Action",
			ExpectedCode: http.StatusOK,
			Message:      "webhooks fetched successfully",
			Method:       http.MethodGet,
			RequestURI:   url.URL{Path: webhook_path + "/all"},
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	for _, test := range tests {
		r := gin.Default()

		webhook := webhook.Controller{Db: db, Validator: validatorRef, Logger: logger}

		webhookUrl := r.Group(fmt.Sprintf("%v/webhooks","/api/v1"), middleware.Authorize(db.Postgresql))
		{
			webhookUrl.GET("/:channel_id/history/:webhook_id", webhook.GetWebhookHistory)
			webhookUrl.GET("/:channel_id/all", webhook.GetAllWebhook)
			webhookUrl.GET("/:channel_id", webhook.GetChannelWebhook)
			webhookUrl.POST("/:channel_id", webhook.CreateWebhook)
			webhookUrl.DELETE("/:channel_id/:webhook_id", webhook.DeleteWebhook)
			webhookUrl.PUT("/:channel_id/:webhook_id", webhook.UpdateWebhook)
			webhookUrl.PUT("/:channel_id/:webhook_id/change-status", webhook.ChangeWebhookStatus)

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
