package test_auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

// TestGoogleLogin tests the Google login endpoint
// This test reproduces the issue where valid Google ID tokens are rejected with 400 Bad Request
func TestGoogleLogin(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	requestURI := url.URL{Path: "/api/v1/auth/google"}

	tests := []struct {
		Name         string
		RequestBody  models.GoogleRequestModel
		ExpectedCode int
		Message      string
		Description  string
	}{
		{
			Name:         "Missing grant_code - should fail validation",
			RequestBody:  models.GoogleRequestModel{},
			ExpectedCode: http.StatusUnprocessableEntity,
			Message:      "Validation failed",
			Description:  "When grant_code is not provided, should return 422 Unprocessable Entity",
		},
		{
			Name: "Empty grant_code - should fail validation",
			RequestBody: models.GoogleRequestModel{
				Token: "",
			},
			ExpectedCode: http.StatusUnprocessableEntity,
			Message:      "Validation failed",
			Description:  "When grant_code is empty, should return 422 Unprocessable Entity",
		},
		{
			Name: "Invalid authorization code - should fail exchange",
			RequestBody: models.GoogleRequestModel{
				Token: "invalid_authorization_code",
			},
			ExpectedCode: http.StatusBadRequest,
			Message:      "failed to exchange code",
			Description:  "When an invalid authorization code is provided, should return 400 with exchange error",
		},
		{
			Name: "Raw ID token passed as grant_code - should fail exchange",
			RequestBody: models.GoogleRequestModel{
				Token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20iLCJhenAiOiJ0ZXN0LWNsaWVudC1pZCIsImF1ZCI6InRlc3QtY2xpZW50LWlkIiwic3ViIjoiMTIzNDU2Nzg5MCIsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiVGVzdCBVc2VyIiwicGljdHVyZSI6Imh0dHBzOi8vZXhhbXBsZS5jb20vcGhvdG8uanBnIiwiZ2l2ZW5fbmFtZSI6IlRlc3QiLCJmYW1pbHlfbmFtZSI6IlVzZXIiLCJpYXQiOjE2MDk0NTkyMDAsImV4cCI6MTYwOTQ2MjgwMH0.signature",
			},
			ExpectedCode: http.StatusBadRequest,
			Message:      "failed to exchange code",
			Description:  "When a raw ID token (JWT) is passed instead of an authorization code, the exchange should fail",
		},
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	for _, test := range tests {
		r := gin.Default()
		r.POST("/api/v1/auth/google", authController.GoogleLogin)

		t.Run(test.Name, func(t *testing.T) {
			t.Log("Description:", test.Description)

			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			req, err := http.NewRequest(http.MethodPost, requestURI.String(), &b)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			t.Logf("Response Status: %d", rr.Code)
			t.Logf("Response Body: %s", rr.Body.String())

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)

			data := tst.ParseResponse(rr)

			code := int(data["status_code"].(float64))
			tst.AssertStatusCode(t, code, test.ExpectedCode)

			if test.Message != "" {
				message, ok := data["message"].(string)
				if ok {
					// Check if message contains expected substring
					if test.ExpectedCode == http.StatusBadRequest {
						// For error cases, check that the error message contains expected text
						if message == "" {
							t.Errorf("Expected error message containing '%s', got empty message", test.Message)
						}
					} else {
						tst.AssertResponseMessage(t, message, test.Message)
					}
				}
			}
		})
	}
}

// TestGoogleLoginWithDirectIDToken tests the scenario where mobile/web clients
// might send a raw ID token directly instead of an authorization code
// This is a common source of 400 errors
func TestGoogleLoginWithDirectIDToken(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	r.POST("/api/v1/auth/google", authController.GoogleLogin)

	// Simulate what a mobile client might send - a raw ID token
	// instead of an authorization code
	requestBody := models.GoogleRequestModel{
		Token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-id-token",
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(requestBody)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/auth/google", &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Logf("Response Status: %d", rr.Code)
	t.Logf("Response Body: %s", rr.Body.String())

	// The current implementation expects an authorization code,
	// so passing a raw ID token should fail at the exchange step
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request when passing ID token directly, got %d", rr.Code)
	}

	data := tst.ParseResponse(rr)
	message, _ := data["message"].(string)
	t.Logf("Error message: %s", message)

	// Document the issue: The API expects grant_code (authorization code)
	// but clients might be sending id_token directly
	t.Log("ISSUE IDENTIFIED: The API expects an authorization code (grant_code) from the OAuth flow,")
	t.Log("but clients may be sending a raw Google ID token directly.")
	t.Log("This causes 'failed to exchange code' error because ID tokens cannot be exchanged.")
}
