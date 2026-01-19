package test_channel

import (
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
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetChannelPreviewMedia(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser_prev_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username_prev_%v", currUUID),
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
		Name:        fmt.Sprintf("TestTeamOfPrev_%s", currUUID),
		Description: "Some Random description",
		Email:       fmt.Sprintf("testuser_prev_%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, org, createOrgData, token)

	createChannelsData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("TestChannelPrev%s", utility.GenerateUUID()),
		Username:       fmt.Sprintf("MrPrevChannels%s", utility.GenerateUUID()),
		OrganisationID: orgId,
		Description:    "Channel for testing preview media",
	}

	channelsId, _ := tst.CreateChannels(t, r, channelController, db, createChannelsData, token)

	// Create a thread with media
	mediaFile := models.File{
		ID:       utility.GenerateUUID(),
		FileName: "test_image.png",
		FileType: "image/png",
		MimeType: "image/png",
		FileLink: "https://example.com/test_image.png",
	}

	threadDoc := models.ThreadDocument{
		ID:         utility.GenerateUUID(),
		ChannelsID: channelsId,
		Content:    "Test message for preview media",
		UserId:     tst.GetUserIDFromToken(t, token, db),
		CreatedAt:  time.Now(),
		Type:       "thread",
		Media:      []models.File{mediaFile},
	}

	err := threadDoc.CreateThread(db, logger)
	if err != nil {
		t.Fatalf("Failed to create test thread with media: %v", err)
	}

	// Allow elasticsearch to index
	time.Sleep(2 * time.Second)

	channelUrl := r.Group(fmt.Sprintf("%v", "/api/v1/channels"), middleware.Authorize(db.Postgresql))
	{
		channelUrl.GET("/:channelId", channelController.GetChannel)
	}

	t.Run("Get Channel with Preview Media", func(t *testing.T) {
		getChannelPath := fmt.Sprintf("/api/v1/channels/%s", channelsId)
		req, err := http.NewRequest(http.MethodGet, getChannelPath, nil)
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
		tst.AssertResponseMessage(t, message, "channel retreived successfully")

		responseData := data["data"].(map[string]interface{})

		if responseData["channels_id"] != channelsId {
			t.Errorf("Expected channel ID %s, got %v", channelsId, responseData["channels_id"])
		}

		previewMedia, exists := responseData["preview_media"]
		if !exists {
			t.Fatal("Expected preview_media field in response")
		}

		previewMediaArray, ok := previewMedia.([]interface{})
		if !ok {
			t.Fatal("Expected preview_media to be an array")
		}

		if len(previewMediaArray) == 0 {
			t.Fatal("Expected at least one media item in preview_media")
		}

		firstMedia := previewMediaArray[0].(map[string]interface{})
		if firstMedia["file_name"] != "test_image.png" {
			t.Errorf("Expected file name 'test_image.png', got %s", firstMedia["file_name"])
		}

		// Assert CreatedAt exists (it might be a string in JSON)
		if _, exists := responseData["created_at"]; !exists {
			t.Errorf("Expected created_at field in response")
		}
	})
}
