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
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
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

	channels_id, _ := tst.CreateChannels(t, r, channelController, db, createChannelsData, token)
	webhook_path := fmt.Sprintf("/api/v1/webhooks/%s", channels_id)

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
			Name:         "Create Webhook Action",
			ExpectedCode: http.StatusBadRequest,
			Message:      "webhook already exists",
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

	defer func() {
		err := tydb.DeleteCollection(db.TypeSense, channels_id)
		if err != nil {
			t.Fatalf("failed to delete collection: %v", err)
		}
		fmt.Printf("deleted collection: %v", channels_id)
	}()

	for _, test := range tests {
		r := gin.Default()

		webhook := webhook.Controller{Db: db, Validator: validatorRef, Logger: logger}

		webhookUrl := r.Group(fmt.Sprintf("%v/webhooks", "/api/v1"), middleware.Authorize(db.Postgresql))
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

func TestChangeWebhookStatus(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("whuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("wh_username%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelController := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	webhookCtrl := webhook.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authCtrl, loginData)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("WHOrg%s", currUUID),
		Description: "Org for webhook status test",
		Email:       userSignUpData.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token)

	createChannelsData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("WHChan%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("whchan%s", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Channel for webhook status test",
	}

	channelID, _ := tst.CreateChannels(t, r, channelController, db, createChannelsData, token)

	defer func() {
		_ = db.Postgresql.Where("channel_id = ?", channelID).Delete(&models.Webhook{}).Error
		_ = db.Postgresql.Where("id = ?", channelID).Delete(&models.Channels{}).Error
		_ = db.Postgresql.Where("id = ?", orgId).Delete(&models.Organisation{}).Error
		_ = db.Postgresql.Where("email = ?", userSignUpData.Email).Delete(&models.User{}).Error
		_ = tydb.DeleteCollection(db.TypeSense, channelID)
	}()

	// Query created webhook from db
	var wh models.Webhook
	err := db.Postgresql.Where("channel_id = ?", channelID).First(&wh).Error
	if err != nil {
		t.Fatalf("failed to find webhook for channel: %v", err)
	}

	rGroup := r.Group("/api/v1/webhooks", middleware.Authorize(db.Postgresql))
	rGroup.PUT("/:channel_id/:webhook_id/change-status", webhookCtrl.ChangeWebhookStatus)

	t.Run("Successfully Change Webhook Status", func(t *testing.T) {
		body := map[string]string{
			"webhook_status": "disabled",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(body)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/webhooks/%s/%s/change-status", channelID, wh.ID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		tst.AssertResponseMessage(t, data["message"].(string), "webhook status updated successfully")
	})

	t.Run("Invalid Webhook ID Format Returns 400", func(t *testing.T) {
		body := map[string]string{
			"webhook_status": "disabled",
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(body)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/webhooks/%s/invalid-uuid/change-status", channelID), &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})
}
