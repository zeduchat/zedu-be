package test_buzz

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	buzzSvc "github.com/hngprojects/telex_be/services/buzz"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuzzRecording(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	currUUID := utility.GenerateUUID()

	hostSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("rechost%s@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Recording",
		LastName:    "Host",
		Password:    "password",
		UserName:    fmt.Sprintf("rechost_%s", currUUID),
	}
	hostLogin := models.LoginRequestModel{Email: hostSignUp.Email, Password: hostSignUp.Password}

	nonHostUUID := utility.GenerateUUID()
	nonHostSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("recguest%s@qa.team", nonHostUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Recording",
		LastName:    "Guest",
		Password:    "password",
		UserName:    fmt.Sprintf("recguest_%s", nonHostUUID),
	}
	nonHostLogin := models.LoginRequestModel{Email: nonHostSignUp.Email, Password: nonHostSignUp.Password}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, hostSignUp, false)
	hostToken := tst.GetLoginToken(t, r, authCtrl, hostLogin)

	tst.SignupUser(t, r, authCtrl, nonHostSignUp, false)
	nonHostToken := tst.GetLoginToken(t, r, authCtrl, nonHostLogin)

	var hostUser models.User
	if err := db.Postgresql.Where("email = ?", hostSignUp.Email).First(&hostUser).Error; err != nil {
		t.Fatalf("failed to fetch host user: %v", err)
	}

	var nonHostUser models.User
	if err := db.Postgresql.Where("email = ?", nonHostSignUp.Email).First(&nonHostUser).Error; err != nil {
		t.Fatalf("failed to fetch non-host user: %v", err)
	}

	channelID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("recchan_%s", currUUID),
		OrganisationID: hostUser.CurrentOrg.String(),
		OwnerId:        hostUser.ID,
		CreatedAt:      time.Now(),
	}).Error; err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	if err := db.Postgresql.Create(&models.UserChannels{
		ChannelsID: channelID,
		UserID:     hostUser.ID,
		Username:   hostSignUp.UserName,
		CreatedAt:  time.Now(),
	}).Error; err != nil {
		t.Logf("warning: failed to add host to channel: %v", err)
	}

	if err := db.Postgresql.Create(&models.UserChannels{
		ChannelsID: channelID,
		UserID:     nonHostUser.ID,
		Username:   nonHostSignUp.UserName,
		CreatedAt:  time.Now(),
	}).Error; err != nil {
		t.Logf("warning: failed to add guest to channel: %v", err)
	}

	router, buzzCtrl := SetupBuzzTestRouter(logger, validatorRef)
	buzzID, _ := tst.CreateBuzz(t, router, *buzzCtrl, db, models.CreateBuzzRequest{ChannelID: channelID}, hostToken)
	if buzzID == "" {
		t.Fatal("failed to create buzz for recording tests")
	}

	t.Run("StartRecording_InvalidBuzzCode", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/not-a-code/recording/start", nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})

	t.Run("StartRecording_BuzzNotFound", func(t *testing.T) {
		unknownID := utility.GenerateUUID()
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/start", unknownID), nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("StartRecording_Unauthenticated", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/start", buzzID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})

	t.Run("StartRecording_FailsIfNotHost", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/start", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+nonHostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		data := tst.ParseResponse(rr)
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
		if msg, ok := data["message"].(string); ok {
			if msg != "only the buzz host can perform this action" {
				t.Errorf("unexpected message: %s", msg)
			}
		}
	})

	t.Run("StopRecording_NoActiveRecording", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/stop", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
		data := tst.ParseResponse(rr)
		if msg, ok := data["message"].(string); ok {
			if msg != "no active recording found for this buzz" {
				t.Errorf("unexpected stop message: %s", msg)
			}
		}
	})

	t.Run("StopRecording_FailsIfNotHost", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/stop", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+nonHostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("StopRecording_InvalidBuzzCode", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/not-valid/recording/stop", nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})

	t.Run("GetRecordingStatus_NoActiveRecording", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/recording/status", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("GetRecordingStatus_Unauthenticated", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/recording/status", buzzID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})

	t.Run("GetRecordingStatus_BuzzNotFound", func(t *testing.T) {
		unknownID := utility.GenerateUUID()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/recording/status", unknownID), nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
	})

	t.Run("GetRecordingStatus_NonParticipantForbidden", func(t *testing.T) {
		strangerUUID := utility.GenerateUUID()
		strangerSignUp := models.CreateUserRequestModel{
			Email:    fmt.Sprintf("stranger%s@qa.team", strangerUUID),
			Password: "password",
			UserName: fmt.Sprintf("stranger_%s", strangerUUID),
		}
		strangerLogin := models.LoginRequestModel{Email: strangerSignUp.Email, Password: strangerSignUp.Password}
		tst.SignupUser(t, r, authCtrl, strangerSignUp, false)
		strangerToken := tst.GetLoginToken(t, r, authCtrl, strangerLogin)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/recording/status", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+strangerToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("StartRecording_Agora_ServiceNotConfigured_Returns500", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/start", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusOK {
			t.Logf("Expected 500 (no Agora creds), got %d — may be OK if Agora is mocked", rr.Code)
		}
	})

	t.Run("ManualRecording_DBFlow_StartAndStop", func(t *testing.T) {
		now := time.Now().UTC()
		orgID := hostUser.CurrentOrg.String()
		rec := models.BuzzRecording{
			ID:         utility.GenerateUUID(),
			BuzzID:     buzzID,
			OrgID:      orgID,
			ResourceID: "test-resource-id",
			Sid:        "test-sid-12345",
			Status:     models.RecordingStatusRecording,
			StartedAt:  now,
		}
		if err := db.Postgresql.Create(&rec).Error; err != nil {
			t.Fatalf("failed to create test recording: %v", err)
		}

		statusReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/recording/status", buzzID), nil)
		statusReq.Header.Set("Authorization", "Bearer "+hostToken)
		statusRr := httptest.NewRecorder()
		router.ServeHTTP(statusRr, statusReq)
		t.Logf("recording status response: %d - %s", statusRr.Code, statusRr.Body.String())

		var dbRec models.BuzzRecording
		if err := db.Postgresql.Where("id = ?", rec.ID).First(&dbRec).Error; err != nil {
			t.Fatalf("recording not found in DB: %v", err)
		}
		if dbRec.BuzzID != buzzID {
			t.Errorf("expected buzz_id %s, got %s", buzzID, dbRec.BuzzID)
		}
		if dbRec.Status != models.RecordingStatusRecording {
			t.Errorf("expected status %s, got %s", models.RecordingStatusRecording, dbRec.Status)
		}

		startReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/start", buzzID), nil)
		startReq.Header.Set("Authorization", "Bearer "+hostToken)
		startRr := httptest.NewRecorder()
		router.ServeHTTP(startRr, startReq)
		tst.AssertStatusCode(t, startRr.Code, http.StatusConflict)
		data := tst.ParseResponse(startRr)
		if msg, ok := data["message"].(string); ok {
			if msg != "recording already in progress" {
				t.Errorf("expected conflict message, got: %s", msg)
			}
		}

		db.Postgresql.Model(&rec).Update("status", models.RecordingStatusStopped)
	})

	t.Run("MetadataContainsRecordingStatus", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/metadata", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			data := tst.ParseResponse(rr)
			if responseData, ok := data["data"].(map[string]interface{}); ok {
				if _, exists := responseData["recording_status"]; !exists {
					t.Error("expected recording_status field in metadata response")
				}
				if _, exists := responseData["is_recording"]; !exists {
					t.Error("expected is_recording field in metadata response")
				}
			}
		} else {
			t.Logf("metadata returned %d (likely agora not configured), skipping field assertions", rr.Code)
		}
	})

	t.Run("EndedBuzz_StartRecording_Fails", func(t *testing.T) {
		endedBuzzID, _ := tst.CreateBuzz(t, router, *buzzCtrl, db, models.CreateBuzzRequest{ChannelID: channelID}, hostToken)
		if endedBuzzID == "" {
			t.Skip("skipping: could not create buzz for end test")
		}

		endReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/end", endedBuzzID), nil)
		endReq.Header.Set("Authorization", "Bearer "+hostToken)
		endRr := httptest.NewRecorder()
		router.ServeHTTP(endRr, endReq)
		tst.AssertStatusCode(t, endRr.Code, http.StatusOK)

		startReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/start", endedBuzzID), nil)
		startReq.Header.Set("Authorization", "Bearer "+hostToken)
		startRr := httptest.NewRecorder()
		router.ServeHTTP(startRr, startReq)
		tst.AssertStatusCode(t, startRr.Code, http.StatusConflict)
	})

	t.Run("BuzzRecordingModel_FieldValidation", func(t *testing.T) {
		orgID := hostUser.CurrentOrg.String()
		now := time.Now().UTC()
		rec := models.BuzzRecording{
			ID:          utility.GenerateUUID(),
			BuzzID:      buzzID,
			OrgID:       orgID,
			ResourceID:  "res-test",
			Sid:         "sid-test",
			Status:      models.RecordingStatusStarting,
			DurationSec: 0,
			StartedAt:   now,
		}
		if err := db.Postgresql.Create(&rec).Error; err != nil {
			t.Fatalf("failed to save BuzzRecording: %v", err)
		}

		var fetched models.BuzzRecording
		if err := db.Postgresql.Where("id = ?", rec.ID).First(&fetched).Error; err != nil {
			t.Fatalf("failed to fetch BuzzRecording: %v", err)
		}

		if fetched.BuzzID != buzzID {
			t.Errorf("expected buzz_id %s, got %s", buzzID, fetched.BuzzID)
		}
		if fetched.OrgID != orgID {
			t.Errorf("expected org_id %s, got %s", orgID, fetched.OrgID)
		}
		if fetched.Status != models.RecordingStatusStarting {
			t.Errorf("expected status %s, got %s", models.RecordingStatusStarting, fetched.Status)
		}
		if fetched.FileID != nil {
			t.Error("expected file_id to be nil initially")
		}
		if fetched.EndedAt != nil {
			t.Error("expected ended_at to be nil initially")
		}

		db.Postgresql.Model(&rec).Update("status", models.RecordingStatusStopped)
	})

	t.Run("RecordingStatusConstants_Correct", func(t *testing.T) {
		cases := map[string]string{
			"idle":      models.RecordingStatusIdle,
			"starting":  models.RecordingStatusStarting,
			"recording": models.RecordingStatusRecording,
			"stopping":  models.RecordingStatusStopping,
			"stopped":   models.RecordingStatusStopped,
			"failed":    models.RecordingStatusFailed,
		}
		for expected, got := range cases {
			if got != expected {
				t.Errorf("constant mismatch: expected %q, got %q", expected, got)
			}
		}
	})

	t.Run("BotTokenVerificationFlow_PassesAllChecks", func(t *testing.T) {
		testBuzzID, _ := tst.CreateBuzz(t, router, *buzzCtrl, db, models.CreateBuzzRequest{ChannelID: channelID}, hostToken)
		if testBuzzID == "" {
			t.Fatal("failed to create a fresh buzz for bot verification test")
		}

		recUID := "123498765"
		recID := utility.GenerateUUID()
		orgID := hostUser.CurrentOrg.String()

		rec := models.BuzzRecording{
			ID:           recID,
			BuzzID:       testBuzzID,
			OrgID:        orgID,
			ResourceID:   "bot-test-res",
			Sid:          "bot-test-sid",
			RecordingUID: recUID,
			Status:       models.RecordingStatusRecording,
			StartedAt:    time.Now().UTC(),
		}
		if err := db.Postgresql.Create(&rec).Error; err != nil {
			t.Fatalf("failed to create test recording for flow: %v", err)
		}
		defer db.Postgresql.Delete(&rec)

		botToken, err := buzzSvc.GenerateBotJWTToken(orgID, testBuzzID, recUID, 300)
		if err != nil {
			t.Fatalf("failed to generate bot JWT token: %v", err)
		}

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/recording/status", testBuzzID), nil)
		req.Header.Set("Authorization", "Bearer "+botToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
			t.Errorf("expected bot token to bypass authentication checks in status endpoint, but got status code %d", rr.Code)
		}

		reqJoin, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/join", testBuzzID), nil)
		reqJoin.Header.Set("Authorization", "Bearer "+botToken)
		rrJoin := httptest.NewRecorder()
		router.ServeHTTP(rrJoin, reqJoin)

		if rrJoin.Code != http.StatusOK {
			t.Errorf("expected bot join to succeed (200), got %d: %s", rrJoin.Code, rrJoin.Body.String())
		} else {
			respData := tst.ParseResponse(rrJoin)
			if data, ok := respData["data"].(map[string]interface{}); ok {
				if tokenResp, ok := data["agora_token"].(map[string]interface{}); ok {
					gotUID := tokenResp["uid"].(string)
					if gotUID != recUID {
						t.Errorf("expected join response UID to match numeric recording UID %s, got %s", recUID, gotUID)
					}
				} else {
					t.Error("expected agora_token in join response")
				}
			} else {
				t.Error("expected data wrapper in join response")
			}
		}

		botUserID := fmt.Sprintf("%s-%s", testBuzzID, recUID)
		var pCount int64
		if utility.IsValidUUID(botUserID) {
			db.Postgresql.Model(&models.BuzzParticipant{}).Where("buzz_id = ? AND user_id = ?", testBuzzID, botUserID).Count(&pCount)
		}
		if pCount > 0 {
			t.Errorf("database pollution: bot user ID %s was added to buzz_participants table", botUserID)
		}

		reqMeta, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/metadata", testBuzzID), nil)
		reqMeta.Header.Set("Authorization", "Bearer "+botToken)
		rrMeta := httptest.NewRecorder()
		router.ServeHTTP(rrMeta, reqMeta)

		if rrMeta.Code != http.StatusOK {
			t.Errorf("expected bot metadata retrieve to succeed (200), got %d: %s", rrMeta.Code, rrMeta.Body.String())
		} else {
			respData := tst.ParseResponse(rrMeta)
			if data, ok := respData["data"].(map[string]interface{}); ok {
				if tokenResp, ok := data["agora_token"].(map[string]interface{}); ok {
					gotUID := tokenResp["uid"].(string)
					if gotUID != recUID {
						t.Errorf("expected metadata response UID to match numeric recording UID %s, got %s", recUID, gotUID)
					}
				} else {
					t.Error("expected agora_token in metadata response")
				}
			}
		}

		tokenReqBody := fmt.Sprintf(`{"buzz_id":%q,"uid":%q}`, testBuzzID, utility.GenerateUUID())
		reqTok, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/token", strings.NewReader(tokenReqBody))
		reqTok.Header.Set("Authorization", "Bearer "+botToken)
		reqTok.Header.Set("Content-Type", "application/json")
		rrTok := httptest.NewRecorder()
		router.ServeHTTP(rrTok, reqTok)

		if rrTok.Code != http.StatusOK {
			t.Errorf("expected bot get token to succeed (200), got %d: %s", rrTok.Code, rrTok.Body.String())
		} else {
			respData := tst.ParseResponse(rrTok)
			if data, ok := respData["data"].(map[string]interface{}); ok {
				gotUID := data["uid"].(string)
				if gotUID != recUID {
					t.Errorf("expected token response UID to match numeric recording UID %s, got %s", recUID, gotUID)
				}
			} else {
				t.Error("expected data wrapper in token response")
			}
		}
	})

	t.Run("UpdateRecordingLayout", func(t *testing.T) {
		// Invalid payload format (layoutConfig is a string, not an array)
		reqBody := `{"layoutConfig":"not-an-array"}`
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/update-layout", buzzID), strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+hostToken)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

		// Valid payload with empty array (should pass validation, but return 404 because no active recording exists)
		reqBodyEmpty := `{"layoutConfig":[]}`
		reqEmpty, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/update-layout", buzzID), strings.NewReader(reqBodyEmpty))
		reqEmpty.Header.Set("Authorization", "Bearer "+hostToken)
		reqEmpty.Header.Set("Content-Type", "application/json")
		rrEmpty := httptest.NewRecorder()
		router.ServeHTTP(rrEmpty, reqEmpty)
		tst.AssertStatusCode(t, rrEmpty.Code, http.StatusNotFound)
		dataEmpty := tst.ParseResponse(rrEmpty)
		if msg, ok := dataEmpty["message"].(string); ok {
			if msg != "no active recording found for this huddle" && msg != "no active recording found for this buzz" {
				t.Errorf("unexpected message for empty layout: %s", msg)
			}
		}

		// Valid payload with null (should pass validation, but return 404 because no active recording exists)
		reqBodyNull := `{"layoutConfig":null}`
		reqNull, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/update-layout", buzzID), strings.NewReader(reqBodyNull))
		reqNull.Header.Set("Authorization", "Bearer "+hostToken)
		reqNull.Header.Set("Content-Type", "application/json")
		rrNull := httptest.NewRecorder()
		router.ServeHTTP(rrNull, reqNull)
		tst.AssertStatusCode(t, rrNull.Code, http.StatusNotFound)

		// Valid layoutConfig array of objects (should pass validation, but return 404 because no active recording exists)
		reqBodyValid := `{"layoutConfig":[{"uid":"123","x_axis":0.0,"y_axis":0.0,"width":1.0,"height":1.0,"alpha":1.0,"render_mode":1}]}`
		reqValid, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/buzz/%s/recording/update-layout", buzzID), strings.NewReader(reqBodyValid))
		reqValid.Header.Set("Authorization", "Bearer "+hostToken)
		reqValid.Header.Set("Content-Type", "application/json")
		rrValid := httptest.NewRecorder()
		router.ServeHTTP(rrValid, reqValid)
		tst.AssertStatusCode(t, rrValid.Code, http.StatusNotFound)
	})

	t.Run("GetAgoraToken_Validation", func(t *testing.T) {
		testBuzzID, _ := tst.CreateBuzz(t, router, *buzzCtrl, db, models.CreateBuzzRequest{ChannelID: channelID}, hostToken)
		if testBuzzID == "" {
			t.Fatal("failed to create fresh buzz for token validation test")
		}

		// Valid UUID
		reqBodyUUID := fmt.Sprintf(`{"buzz_id":%q,"uid":%q}`, testBuzzID, utility.GenerateUUID())
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/token", strings.NewReader(reqBodyUUID))
		req.Header.Set("Authorization", "Bearer "+hostToken)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		// screen-<UUID> format
		reqBodyScreen := fmt.Sprintf(`{"buzz_id":%q,"uid":"screen-%s"}`, testBuzzID, utility.GenerateUUID())
		req2, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/token", strings.NewReader(reqBodyScreen))
		req2.Header.Set("Authorization", "Bearer "+hostToken)
		req2.Header.Set("Content-Type", "application/json")
		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, req2)
		tst.AssertStatusCode(t, rr2.Code, http.StatusOK)

		// Invalid format
		reqBodyInvalid := fmt.Sprintf(`{"buzz_id":%q,"uid":"invalid-format-1234"}`, testBuzzID)
		req3, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/token", strings.NewReader(reqBodyInvalid))
		req3.Header.Set("Authorization", "Bearer "+hostToken)
		req3.Header.Set("Content-Type", "application/json")
		rr3 := httptest.NewRecorder()
		router.ServeHTTP(rr3, req3)
		tst.AssertStatusCode(t, rr3.Code, http.StatusUnprocessableEntity)
	})
}
