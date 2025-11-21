package test_huddle

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestHuddle(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authController := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelController := channel.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	router, _ := SetupHuddlesTestRouter()

	userEmail := utility.GenerateUUID() + "@qa.team"
	signUp := models.CreateUserRequestModel{Email: userEmail, Password: "password"}
	login := models.LoginRequestModel{Email: signUp.Email, Password: signUp.Password}

	tst.SignupUser(t, router, authController, signUp, false)
	token := tst.GetLoginToken(t, router, authController, login)
	if token == "" {
		t.Fatalf("failed to obtain login token")
	}

	var user models.User
	if err := db.Postgresql.Where("email = ?", signUp.Email).First(&user).Error; err != nil {
		t.Fatalf("failed to fetch created user: %v", err)
	}
	channelData := models.CreateChannelsRequest{
		OrganisationID: user.CurrentOrg.String(),
		Username:       user.Profile.UserName,
		Name:           "test_" + utility.GenerateUUID(),
	}
	channelID, _ := tst.CreateChannels(t, router, channelController, db, channelData, token)
	if channelID == "" {
		t.Fatal("failed to obtain channelID")
	}
	t.Run("CreateHuddleSuccess", func(t *testing.T) {
		reqBody := models.CreateHuddleRequest{
			ChannelID:      channelID,
			OrganisationID: user.CurrentOrg.String(),
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)
		req, err := http.NewRequest(http.MethodPost, "/api/v1/huddles/create", &b)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		dataM, ok := data["data"].(map[string]any)
		if !ok {
			t.Fatal("expected data to be a map")
		}
		huddleID, ok := dataM["huddle_id"].(string)
		if !ok || huddleID == "" {
			t.Errorf("expected huddle_id to be non-empty")
		}
		hostID, ok := dataM["host_id"].(string)
		if !ok || hostID == "" {
			t.Errorf("expected host_id to be non-empty")
		}
		createdAt, ok := dataM["created_at"].(string)
		if !ok || createdAt == "" {
			t.Errorf("expected created_at to be non-empty")
		}

		if hostID != user.ID {
			t.Errorf("expected %s to be hostID but got %s", user.ID, hostID)
		}

		participantIDs, ok := dataM["participants_id"].([]any)
		if !ok {
			t.Fatalf("expected participants_id to be an array, got %T", dataM["participants_id"])
		}
		if len(participantIDs) < 1 {
			t.Errorf("expected host to be a participant")
		}

	})

	t.Run("InvalidChannelIDReturns404", func(t *testing.T) {
		reqBody := models.CreateHuddleRequest{
			ChannelID:      utility.GenerateUUID(),
			OrganisationID: user.CurrentOrg.String(),
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)
		req, err := http.NewRequest(http.MethodPost, "/api/v1/huddles/create", &b)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("InvalidOrganisationIDReturns404", func(t *testing.T) {
		reqBody := models.CreateHuddleRequest{
			ChannelID:      channelID,
			OrganisationID: utility.GenerateUUID(),
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)
		req, err := http.NewRequest(http.MethodPost, "/api/v1/huddles/create", &b)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

}
