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

func TestBuzzNotes(t *testing.T) {
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

	buzzID := utility.GenerateUUID()
	h := models.Buzz{ID: buzzID, ChannelID: utility.GenerateUUID(), HostID: user.ID, ParticipantIDs: pq.StringArray{user.ID}, BuzzStartTime: time.Now().UTC()}
	if err := db.Postgresql.Create(&h).Error; err != nil {
		t.Fatalf("failed to create buzz: %v", err)
	}

	hp := models.BuzzParticipant{BuzzID: buzzID, UserID: user.ID}
	if err := db.Postgresql.Create(&hp).Error; err != nil {
		t.Fatalf("failed to create buzz participant: %v", err)
	}

	var noteID string

	t.Run("CreateNoteSuccess", func(t *testing.T) {
		reqBody := models.CreateBuzzNoteRequest{Note: "This is a test note"}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/notes", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		noteID = data["id"].(string)
		tst.AssertResponseMessage(t, resp["message"].(string), "buzz note created successfully")
	})

	t.Run("CreateNoteValidationFailed", func(t *testing.T) {
		reqBody := models.CreateBuzzNoteRequest{Note: ""}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/notes", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusUnprocessableEntity)
	})

	t.Run("GetNotesSuccess", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/buzz/"+buzzID+"/notes", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		notes := data["notes"].([]interface{})
		if len(notes) == 0 {
			t.Errorf("expected notes, got empty list")
		}
	})

	t.Run("UpdateNoteSuccess", func(t *testing.T) {
		reqBody := models.UpdateBuzzNoteRequest{Note: "Updated note content"}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/buzz/"+buzzID+"/notes/"+noteID, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		tst.AssertResponseMessage(t, resp["message"].(string), "buzz note updated successfully")
	})

	t.Run("UpdateNoteForbidden", func(t *testing.T) {
		otherEmail := utility.GenerateUUID() + "@qa.team"
		otherSign := models.CreateUserRequestModel{Email: otherEmail, Password: "password"}
		otherLogin := models.LoginRequestModel{Email: otherSign.Email, Password: otherSign.Password}

		// Use a temporary router for the second user signup/login to avoid route collision
		tempRouter := gin.New()
		tst.SignupUser(t, tempRouter, authController, otherSign, false)
		otherToken := tst.GetLoginToken(t, tempRouter, authController, otherLogin)

		// Add other user to buzz so they are a participant (otherwise they get 403 for not being participant)
		var otherUser models.User
		db.Postgresql.Where("email = ?", otherEmail).First(&otherUser)
		hp := models.BuzzParticipant{BuzzID: buzzID, UserID: otherUser.ID}
		db.Postgresql.Create(&hp)

		reqBody := models.UpdateBuzzNoteRequest{Note: "Malicious update"}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/buzz/"+buzzID+"/notes/"+noteID, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+otherToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})
}
