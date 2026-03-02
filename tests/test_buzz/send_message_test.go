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

func TestSendBuzzMessage(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()
	validatorRef := validator.New()
	router, _ := SetupBuzzTestRouter(logger, validatorRef)

	authController := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

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

	var profile models.Profile
	if err := db.Postgresql.Where("userid = ?", user.ID).First(&profile).Error; err != nil {
		t.Fatalf("failed to fetch user profile: %v", err)
	}

	channelID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()
	buzzID := utility.GenerateUUID()
	h := models.Buzz{
		ID:             buzzID,
		ChannelID:      channelID,
		HostID:         user.ID,
		OrgID:          &orgID,
		ParticipantIDs: pq.StringArray{user.ID},
		BuzzStartTime:  time.Now().UTC(),
		Status:         models.BuzzStatusActive,
		OriginalHostID: user.ID,
	}
	if err := db.Postgresql.Create(&h).Error; err != nil {
		t.Fatalf("failed to create buzz: %v", err)
	}

	hp := models.BuzzParticipant{BuzzID: buzzID, UserID: user.ID}
	if err := db.Postgresql.Create(&hp).Error; err != nil {
		t.Fatalf("failed to create buzz participant: %v", err)
	}

	t.Run("SendMessageSuccess", func(t *testing.T) {
		reqBody := models.SendBuzzMessageRequest{
			Content: "Hello from buzz!",
			Media:   []models.File{},
		}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/message", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		tst.AssertResponseMessage(t, resp["message"].(string), "message sent successfully")

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected data to be a map, got %T", resp["data"])
		}
		if msg, ok := data["message"].(string); !ok || msg != "Hello from buzz!" {
			t.Errorf("expected content 'Hello from buzz!', got %v", data["message"])
		}
		if msgType, ok := data["type"].(string); !ok || msgType != "buzz_message" {
			t.Errorf("expected type 'buzz_message', got %v", data["type"])
		}
		if uID, ok := data["user_id"].(string); !ok || uID != user.ID {
			t.Errorf("expected user_id %s, got %v", user.ID, data["user_id"])
		}
	})

	t.Run("SendMessageWithMedia", func(t *testing.T) {
		reqBody := models.SendBuzzMessageRequest{
			Content: "Message with media",
			Media: []models.File{
				{
					FileName: "test.jpg",
					FileLink: "https://example.com/test.jpg",
					FileType: "image",
					MimeType: "image/jpeg",
					Size:     1024,
				},
			},
		}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/message", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected data to be a map, got %T", resp["data"])
		}
		media, ok := data["media"].([]interface{})
		if !ok {
			t.Fatalf("expected media to be a slice, got %T", data["media"])
		}
		if len(media) != 1 {
			t.Errorf("expected 1 media item, got %d", len(media))
		}
	})

	t.Run("SendMessageValidationFailed", func(t *testing.T) {
		reqBody := models.SendBuzzMessageRequest{Content: ""}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/message", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusUnprocessableEntity)
	})

	t.Run("SendMessageInvalidBuzzID", func(t *testing.T) {
		reqBody := models.SendBuzzMessageRequest{Content: "Test message"}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/invalid-id/message", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})

	t.Run("SendMessageBuzzNotFound", func(t *testing.T) {
		nonExistentBuzzID := utility.GenerateUUID()
		reqBody := models.SendBuzzMessageRequest{Content: "Test message"}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+nonExistentBuzzID+"/message", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("SendMessageNotParticipant", func(t *testing.T) {
		otherEmail := utility.GenerateUUID() + "@qa.team"
		otherSign := models.CreateUserRequestModel{Email: otherEmail, Password: "password"}
		otherLogin := models.LoginRequestModel{Email: otherSign.Email, Password: otherSign.Password}

		tempRouter := gin.New()
		tst.SignupUser(t, tempRouter, authController, otherSign, false)
		otherToken := tst.GetLoginToken(t, tempRouter, authController, otherLogin)

		reqBody := models.SendBuzzMessageRequest{Content: "Unauthorized message"}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/message", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+otherToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("SendMessageBuzzEnded", func(t *testing.T) {
		endedBuzzID := utility.GenerateUUID()
		endedBuzz := models.Buzz{
			ID:             endedBuzzID,
			ChannelID:      channelID,
			HostID:         user.ID,
			OrgID:          &orgID,
			ParticipantIDs: pq.StringArray{user.ID},
			BuzzStartTime:  time.Now().UTC(),
			Status:         models.BuzzStatusEnded,
			OriginalHostID: user.ID,
		}
		db.Postgresql.Create(&endedBuzz)

		endedHP := models.BuzzParticipant{BuzzID: endedBuzzID, UserID: user.ID}
		db.Postgresql.Create(&endedHP)

		reqBody := models.SendBuzzMessageRequest{Content: "Message to ended buzz"}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+endedBuzzID+"/message", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})
}
