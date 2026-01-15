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
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuzz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	// Register cleanup to close connections after test completes
	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	authController := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	router, _ := SetupBuzzTestRouter(logger, validatorRef)

	userEmail := utility.GenerateUUID() + "@qa.team"
	signUp := models.CreateUserRequestModel{Email: userEmail, Password: "password", UserName: fmt.Sprintf("user_%s", utility.GenerateUUID())}
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

	// Manually create channel and add user (bypass Elasticsearch issue)
	channelID := utility.GenerateUUID()

	// Create channel directly in database
	channel := models.Channels{
		ID:             channelID,
		Name:           "test_" + utility.GenerateUUID(),
		OrganisationID: user.CurrentOrg.String(),
		OwnerId:        user.ID,
		CreatedAt:      time.Now(),
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	// Add user to channel
	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user.ID,
		Username:   signUp.UserName,
		CreatedAt:  time.Now(),
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Logf("Warning: Failed to add user to channel: %v", err)
	}
	t.Run("CreateBuzzSuccess", func(t *testing.T) {
		reqBody := models.CreateBuzzRequest{
			ChannelID: channelID,
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)
		req, err := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
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
		buzzID, ok := dataM["buzz_id"].(string)
		if !ok || buzzID == "" {
			t.Errorf("expected buzz_id to be non-empty")
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
		reqBody := models.CreateBuzzRequest{
			ChannelID: utility.GenerateUUID(),
		}
		var b bytes.Buffer
		json.NewEncoder(&b).Encode(reqBody)
		req, err := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
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
