package test_buzz

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestSendBuzzReaction(t *testing.T) {
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

	// Create an active buzz
	buzzID := utility.GenerateUUID()
	buzz := models.Buzz{
		ID:             buzzID,
		ChannelID:      utility.GenerateUUID(),
		HostID:         user.ID,
		ParticipantIDs: pq.StringArray{user.ID},
		IsLiveStatus:   true,
		Status:         models.BuzzStatusActive,
	}
	if err := db.Postgresql.Create(&buzz).Error; err != nil {
		t.Fatalf("failed to create test buzz: %v", err)
	}

	// Create participant record
	participant := models.BuzzParticipant{
		ID:     utility.GenerateUUID(),
		BuzzID: buzzID,
		UserID: user.ID,
		Status: models.BuzzParticipantStatusActive,
	}
	if err := db.Postgresql.Create(&participant).Error; err != nil {
		t.Fatalf("failed to create test participant: %v", err)
	}

	t.Run("SendEmojiReaction", func(t *testing.T) {
		reactionReq := models.SendBuzzReactionRequest{
			BuzzID:       buzzID,
			ReactionType: "emoji",
			Content:      "👍",
		}

		body, _ := json.Marshal(reactionReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/reaction", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response["message"] != "reaction sent successfully" {
			t.Errorf("Expected success message, got: %v", response["message"])
		}
	})

	t.Run("SendEffectReaction", func(t *testing.T) {
		reactionReq := models.SendBuzzReactionRequest{
			BuzzID:       buzzID,
			ReactionType: "effect",
			Content:      "fireworks",
		}

		body, _ := json.Marshal(reactionReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/reaction", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("SendGifReaction", func(t *testing.T) {
		reactionReq := models.SendBuzzReactionRequest{
			BuzzID:       buzzID,
			ReactionType: "gif",
			Content:      "https://media.giphy.com/media/example.gif",
		}

		body, _ := json.Marshal(reactionReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/reaction", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("SendReactionInvalidType", func(t *testing.T) {
		reactionReq := map[string]interface{}{
			"buzz_id":       buzzID,
			"reaction_type": "invalid",
			"content":       "test",
		}

		body, _ := json.Marshal(reactionReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/reaction", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", w.Code)
		}
	})

	t.Run("SendReactionNotParticipant", func(t *testing.T) {
		// Create another user who is not a participant
		otherEmail := utility.GenerateUUID() + "@qa.team"
		otherSignUp := models.CreateUserRequestModel{Email: otherEmail, Password: "password"}
		otherLogin := models.LoginRequestModel{Email: otherSignUp.Email, Password: otherSignUp.Password}

		tst.SignupUser(t, router, authController, otherSignUp, false)
		otherToken := tst.GetLoginToken(t, router, authController, otherLogin)

		reactionReq := models.SendBuzzReactionRequest{
			BuzzID:       buzzID,
			ReactionType: "emoji",
			Content:      "👍",
		}

		body, _ := json.Marshal(reactionReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/reaction", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+otherToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	// Cleanup
	db.Postgresql.Delete(&participant)
	db.Postgresql.Delete(&buzz)
	db.Postgresql.Delete(&user)
}

func TestUpdateBuzzSticker(t *testing.T) {
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

	// Create an active buzz
	buzzID := utility.GenerateUUID()
	buzz := models.Buzz{
		ID:             buzzID,
		ChannelID:      utility.GenerateUUID(),
		HostID:         user.ID,
		ParticipantIDs: pq.StringArray{user.ID},
		IsLiveStatus:   true,
		Status:         models.BuzzStatusActive,
	}
	if err := db.Postgresql.Create(&buzz).Error; err != nil {
		t.Fatalf("failed to create test buzz: %v", err)
	}

	// Create participant record
	participantID := utility.GenerateUUID()
	participant := models.BuzzParticipant{
		ID:     participantID,
		BuzzID: buzzID,
		UserID: user.ID,
		Status: models.BuzzParticipantStatusActive,
	}
	if err := db.Postgresql.Create(&participant).Error; err != nil {
		t.Fatalf("failed to create test participant: %v", err)
	}

	t.Run("SetRaiseHandSticker", func(t *testing.T) {
		sticker := "raise_hand"
		stickerReq := models.BuzzStickerUpdateRequest{
			BuzzID:  buzzID,
			Sticker: &sticker,
		}

		body, _ := json.Marshal(stickerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/sticker", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		data := response["data"].(map[string]interface{})
		if data["sticker"] != "raise_hand" {
			t.Errorf("Expected sticker 'raise_hand', got: %v", data["sticker"])
		}

		// Verify in database
		var updatedParticipant models.BuzzParticipant
		if err := db.Postgresql.Where("id = ?", participantID).First(&updatedParticipant).Error; err != nil {
			t.Fatalf("failed to fetch updated participant: %v", err)
		}

		if updatedParticipant.StatusSticker == nil || *updatedParticipant.StatusSticker != "raise_hand" {
			t.Errorf("Expected sticker 'raise_hand' in DB, got: %v", updatedParticipant.StatusSticker)
		}

		if updatedParticipant.StickerSetAt == nil {
			t.Error("Expected sticker_set_at to be set")
		}
	})

	t.Run("SetBRBSticker", func(t *testing.T) {
		sticker := "brb"
		stickerReq := models.BuzzStickerUpdateRequest{
			BuzzID:  buzzID,
			Sticker: &sticker,
		}

		body, _ := json.Marshal(stickerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/sticker", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ClearSticker", func(t *testing.T) {
		stickerReq := models.BuzzStickerUpdateRequest{
			BuzzID:  buzzID,
			Sticker: nil,
		}

		body, _ := json.Marshal(stickerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/sticker", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		// Verify sticker is cleared in database
		var updatedParticipant models.BuzzParticipant
		if err := db.Postgresql.Where("id = ?", participantID).First(&updatedParticipant).Error; err != nil {
			t.Fatalf("failed to fetch updated participant: %v", err)
		}

		if updatedParticipant.StatusSticker != nil {
			t.Errorf("Expected sticker to be nil, got: %v", *updatedParticipant.StatusSticker)
		}

		if updatedParticipant.StickerSetAt != nil {
			t.Error("Expected sticker_set_at to be nil when cleared")
		}
	})

	t.Run("SetStickerInvalidType", func(t *testing.T) {
		sticker := "invalid_sticker"
		stickerReq := map[string]interface{}{
			"buzz_id": buzzID,
			"sticker": sticker,
		}

		body, _ := json.Marshal(stickerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/sticker", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", w.Code)
		}
	})

	t.Run("SetStickerNotParticipant", func(t *testing.T) {
		// Create another user who is not a participant
		otherEmail := utility.GenerateUUID() + "@qa.team"
		otherSignUp := models.CreateUserRequestModel{Email: otherEmail, Password: "password"}
		otherLogin := models.LoginRequestModel{Email: otherSignUp.Email, Password: otherSignUp.Password}

		tst.SignupUser(t, router, authController, otherSignUp, false)
		otherToken := tst.GetLoginToken(t, router, authController, otherLogin)

		sticker := "raise_hand"
		stickerReq := models.BuzzStickerUpdateRequest{
			BuzzID:  buzzID,
			Sticker: &sticker,
		}

		body, _ := json.Marshal(stickerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/sticker", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+otherToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	// Cleanup
	db.Postgresql.Delete(&participant)
	db.Postgresql.Delete(&buzz)
	db.Postgresql.Delete(&user)
}
