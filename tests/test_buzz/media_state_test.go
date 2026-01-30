package test_buzz

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateMediaStateBase(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authController := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	router, _ := SetupBuzzTestRouter(logger, validatorRef)

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

	channelID := utility.GenerateUUID()
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

	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user.ID,
		Username:   "testuser",
		CreatedAt:  time.Now(),
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Logf("Warning: Failed to add user to channel: %v", err)
	}

	buzzID := utility.GenerateUUID()
	h := models.Buzz{ID: buzzID, ChannelID: channelID, HostID: user.ID, ParticipantIDs: pq.StringArray{user.ID}, BuzzStartTime: time.Now().UTC(), Status: models.BuzzStatusActive, IsLiveStatus: true}
	if err := db.Postgresql.Create(&h).Error; err != nil {
		t.Fatalf("failed to create buzz: %v", err)
	}

	hp := models.BuzzParticipant{BuzzID: buzzID, UserID: user.ID, Status: models.BuzzParticipantStatusActive}
	if err := db.Postgresql.Create(&hp).Error; err != nil {
		t.Fatalf("failed to create buzz participant: %v", err)
	}

	t.Run("UpdateMediaStateSuccess", func(t *testing.T) {
		mediaState := map[string]interface{}{"audio": true, "video": false}
		reqBody := models.UpdateMediaStateRequest{MediaState: mediaState}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/buzz/"+buzzID+"/media-state", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		// Verify in DB
		var hpUpdated models.BuzzParticipant
		db.Postgresql.Where("buzz_id = ? AND user_id = ?", buzzID, user.ID).First(&hpUpdated)
		if hpUpdated.MediaState == nil {
			t.Errorf("expected media state to be updated in DB")
		}
	})

	t.Run("GetMetadataIncludesMediaState", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/buzz/"+buzzID+"/metadata", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		participants := data["participants"].([]interface{})

		found := false
		for _, p := range participants {
			pm := p.(map[string]interface{})
			if pm["user_id"] == user.ID {
				if pm["media_state"] != nil {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("media_state not found in metadata response")
		}
	})

	t.Run("UpdateMediaStateWithInvalidBuzzID", func(t *testing.T) {
		invalidBuzzID := utility.GenerateUUID()
		mediaState := map[string]interface{}{"audio": true, "video": false}
		reqBody := models.UpdateMediaStateRequest{MediaState: mediaState}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/buzz/"+invalidBuzzID+"/media-state", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("GetMetadataWithInvalidBuzzID", func(t *testing.T) {
		invalidBuzzID := utility.GenerateUUID()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/buzz/"+invalidBuzzID+"/metadata", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})
}
