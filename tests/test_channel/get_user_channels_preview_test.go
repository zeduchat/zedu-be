package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

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

func TestGetUserChannelsWithPreviewThread(t *testing.T) {
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

	auth := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

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

	channelsId, _ := tst.CreateChannels(t, r, channelController, db, createChannelsData, token)

	threadDoc := models.ThreadDocument{
		ID:         utility.GenerateUUID(),
		ChannelsID: channelsId,
		Content:    "Test message for preview",
		UserId:     tst.GetUserIDFromToken(t, token, db),
		CreatedAt:  time.Now(),
		Type:       "thread",
	}

	err := threadDoc.CreateThread(db, logger)
	if err != nil {
		t.Fatalf("Failed to create test thread: %v", err)
	}

	t.Run("Get User Channels with Preview Thread", func(t *testing.T) {
		r := gin.Default()

		channelUrl := r.Group(fmt.Sprintf("%v", "/api/v1/channels"), middleware.Authorize(db.Postgresql))
		{
			channelUrl.GET("/org/:org_id", channelController.GetUserChannels)
		}

		getUserChannelsPath := fmt.Sprintf("/api/v1/channels/org/%s", orgId)
		getUserChannelsURI := url.URL{Path: getUserChannelsPath}

		req, err := http.NewRequest(http.MethodGet, getUserChannelsURI.String(), nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)

		code := int(data["status_code"].(float64))
		tst.AssertStatusCode(t, code, http.StatusOK)

		message := data["message"].(string)
		tst.AssertResponseMessage(t, message, "user channels fetched successfully")

		responseData := data["data"].([]interface{})
		if len(responseData) == 0 {
			t.Fatal("Expected at least one channel in response")
		}

		channelData := responseData[0].(map[string]interface{})

		if channelData["channels_id"] != channelsId {
			t.Errorf("Expected channel ID %s, got %s", channelsId, channelData["channels_id"])
		}

		previewThread, exists := channelData["preview_thread"]
		if !exists {
			t.Fatal("Expected preview_thread field in response")
		}

		previewThreadArray, ok := previewThread.([]interface{})
		if !ok {
			t.Fatal("Expected preview_thread to be an array")
		}

		if len(previewThreadArray) == 0 {
			t.Fatal("Expected at least one thread in preview_thread")
		}

		firstThread := previewThreadArray[0].(map[string]interface{})
		if firstThread["message"] != "Test message for preview" {
			t.Errorf("Expected thread content 'Test message for preview', got %s", firstThread["message"])
		}
	})

	t.Run("Verify Channels Sorted by Preview Thread Created At", func(t *testing.T) {
		createChannelsData2 := models.CreateChannelsRequest{
			Name:           fmt.Sprintf("TestChannels2%s", utility.GenerateUUID()),
			Username:       fmt.Sprintf("Mr%sChannels2", utility.GenerateUUID()),
			OrganisationID: orgId,
			Description:    "Second test channel",
		}

		channelsId2, _ := tst.CreateChannels(t, r, channelController, db, createChannelsData2, token)

		threadDoc2 := models.ThreadDocument{
			ID:         utility.GenerateUUID(),
			ChannelsID: channelsId2,
			Content:    "Newer message for preview",
			UserId:     tst.GetUserIDFromToken(t, token, db),
			CreatedAt:  time.Now(),
			Type:       "thread",
		}

		err := threadDoc2.CreateThread(db, logger)
		if err != nil {
			t.Fatalf("Failed to create second test thread: %v", err)
		}

		r := gin.Default()

		channelUrl := r.Group(fmt.Sprintf("%v", "/api/v1/channels"), middleware.Authorize(db.Postgresql))
		{
			channelUrl.GET("/org/:org_id", channelController.GetUserChannels)
		}

		getUserChannelsPath := fmt.Sprintf("/api/v1/channels/org/%s", orgId)
		getUserChannelsURI := url.URL{Path: getUserChannelsPath}

		req, err := http.NewRequest(http.MethodGet, getUserChannelsURI.String(), nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		responseData := data["data"].([]interface{})

		if len(responseData) < 2 {
			t.Fatal("Expected at least two channels in response")
		}

		firstChannel := responseData[0].(map[string]interface{})
		if firstChannel["channels_id"] != channelsId2 {
			t.Errorf("Expected first channel to be %s (newest), got %s", channelsId2, firstChannel["channels_id"])
		}
	})
}
