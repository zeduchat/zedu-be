package test_agora

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/agora"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGenerateRTCToken(t *testing.T) {
	// Ensure required env vars are present for token generation.
	_ = os.Setenv("AGORA_APP_ID", "test-app-id")
	_ = os.Setenv("AGORA_APP_CERTIFICATE", "test-app-cert")

	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validate := validator.New()
	db := storage.Connection()

	authController := auth.Controller{
		Db:        db,
		Validator: validate,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}
	agoraController := &agora.Controller{
		Db:        db,
		Validator: validate,
		Logger:    logger,
	}

	r := gin.Default()
	agoraGroup := r.Group("/api/v1/agora", middleware.Authorize(db.Postgresql))
	{
		agoraGroup.POST("/rtc-token", agoraController.GenerateRTC)
	}

	// Auth setup
	userEmail := utility.GenerateUUID() + "@qa.team"
	signUp := models.CreateUserRequestModel{Email: userEmail, Password: "password"}
	login := models.LoginRequestModel{Email: signUp.Email, Password: signUp.Password}

	tst.SignupUser(t, r, authController, signUp, false)
	token := tst.GetLoginToken(t, r, authController, login)
	if token == "" {
		t.Fatalf("failed to obtain login token")
	}

	// Successful RTC token generation
	body := models.AgoraTokenRequest{
		ChannelName: "demo-channel",
		Role:        "publisher",
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/api/v1/agora/rtc-token", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	resp := tst.ParseResponse(rr)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a map in response")
	}
	rtcToken, ok := data["rtc_token"].(string)
	if !ok || rtcToken == "" {
		t.Fatalf("expected rtc_token to be present")
	}

	// Validation failure when channel_name is missing
	invalidBody := models.AgoraTokenRequest{
		Role: "publisher",
	}
	buf.Reset()
	_ = json.NewEncoder(&buf).Encode(invalidBody)
	reqInvalid, _ := http.NewRequest(http.MethodPost, "/api/v1/agora/rtc-token", &buf)
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.Header.Set("Authorization", "Bearer "+token)

	rrInvalid := httptest.NewRecorder()
	r.ServeHTTP(rrInvalid, reqInvalid)
	if rrInvalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for validation error, got %d", rrInvalid.Code)
	}
}
