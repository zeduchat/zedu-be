package test_account

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreateAccountDeletionRequest(t *testing.T) {
	_, accountController := SetupAccountTestRouter()
	db := accountController.Db.Postgresql
	currUUID := utility.GenerateUUID()

	setup := func() (*gin.Engine, *auth.Controller, string, models.User) {
		router, accountController := SetupAccountTestRouter()
		authController := auth.Controller{
			Db:        accountController.Db,
			Validator: accountController.Validator,
			Logger:    accountController.Logger,
		}

		password, _ := utility.HashPassword("password")
		orgUUID, _ := uuid.FromString(utility.GenerateUUID())
		user := models.User{
			ID:         utility.GenerateUUID(),
			Name:       "Test User",
			Email:      fmt.Sprintf("testuser%v@qa.team", currUUID),
			Password:   password,
			IsActive:   true,
			IsVerified: true,
			CurrentOrg: orgUUID,
		}
		db.Create(&user)

		loginData := models.LoginRequestModel{
			Email:    user.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, authController, loginData)

		return router, &authController, token, user
	}

	t.Run("Successful Account Deletion Request", func(t *testing.T) {
		router, _, token, user := setup()

		requestData := map[string]string{
			"fullname":        "John Doe",
			"email":           fmt.Sprintf("john%v@example.com", currUUID),
			"reason":          "Privacy concerns",
			"additional_info": "I want my data deleted",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Account deletion request submitted successfully")

		data := response["data"].(map[string]interface{})
		if data["fullname"] != "John Doe" {
			t.Errorf("Expected fullname 'John Doe', got '%v'", data["fullname"])
		}
		if data["email"] != requestData["email"] {
			t.Errorf("Expected email '%s', got '%v'", requestData["email"], data["email"])
		}
		if data["reason"] != "Privacy concerns" {
			t.Errorf("Expected reason 'Privacy concerns', got '%v'", data["reason"])
		}

		var deletionRequest models.AccountDeletionRequest
		db.Where("email = ? AND org_id = ?", requestData["email"], user.CurrentOrg.String()).First(&deletionRequest)
		if deletionRequest.Email != requestData["email"] {
			t.Errorf("Database: Expected email %s, got %s", requestData["email"], deletionRequest.Email)
		}
		if deletionRequest.OrgID != user.CurrentOrg.String() {
			t.Errorf("Database: Expected org_id %s, got %s", user.CurrentOrg.String(), deletionRequest.OrgID)
		}
	})

	t.Run("Successful Request Without Optional Fields", func(t *testing.T) {
		router, _, token, _ := setup()

		requestData := map[string]string{
			"fullname": "Jane Smith",
			"email":    fmt.Sprintf("jane%v@example.com", currUUID),
			"reason":   "No longer need the account",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Account deletion request submitted successfully")
	})

	t.Run("Missing Fullname - Validation Error", func(t *testing.T) {
		router, _, token, _ := setup()

		requestData := map[string]string{
			"email":  fmt.Sprintf("test%v@example.com", currUUID),
			"reason": "Testing validation",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})

	t.Run("Missing Email - Validation Error", func(t *testing.T) {
		router, _, token, _ := setup()

		requestData := map[string]string{
			"fullname": "Test User",
			"reason":   "Testing validation",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})

	t.Run("Missing Reason - Validation Error", func(t *testing.T) {
		router, _, token, _ := setup()

		requestData := map[string]string{
			"fullname": "Test User",
			"email":    fmt.Sprintf("test%v@example.com", currUUID),
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})

	t.Run("Invalid Email Format - Validation Error", func(t *testing.T) {
		router, _, token, _ := setup()

		requestData := map[string]string{
			"fullname": "Test User",
			"email":    "invalid-email-format",
			"reason":   "Testing validation",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})

	t.Run("Missing Authentication Token", func(t *testing.T) {
		router, _, _, _ := setup()

		requestData := map[string]string{
			"fullname": "Test User",
			"email":    fmt.Sprintf("test%v@example.com", currUUID),
			"reason":   "Testing auth",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
	})

	t.Run("Invalid Authentication Token", func(t *testing.T) {
		router, _, _, _ := setup()

		requestData := map[string]string{
			"fullname": "Test User",
			"email":    fmt.Sprintf("test%v@example.com", currUUID),
			"reason":   "Testing auth",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid_token_here")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		router, _, token, _ := setup()

		malformedJSON := []byte(`{"fullname": "Test", "email": "test@example.com"`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(malformedJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Failed to parse request body")
	})

	t.Run("Empty Request Body", func(t *testing.T) {
		router, _, token, _ := setup()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})

	t.Run("Verify OrgId Is Extracted From Token", func(t *testing.T) {
		router, authController, _, _ := setup()

		password, _ := utility.HashPassword("password")
		orgUUID, _ := uuid.FromString(utility.GenerateUUID())
		testUser := models.User{
			ID:         utility.GenerateUUID(),
			Name:       "Org Test User",
			Email:      fmt.Sprintf("orgtest%v@qa.team", currUUID),
			Password:   password,
			IsActive:   true,
			IsVerified: true,
			CurrentOrg: orgUUID,
		}
		db.Create(&testUser)

		loginData := models.LoginRequestModel{
			Email:    testUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		requestData := map[string]string{
			"fullname": "Org Test User",
			"email":    fmt.Sprintf("orgtest%v@example.com", currUUID),
			"reason":   "Testing org_id extraction",
		}
		payload, _ := json.Marshal(requestData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/delete-account", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)

		var deletionRequest models.AccountDeletionRequest
		db.Where("email = ?", requestData["email"]).First(&deletionRequest)

		if deletionRequest.OrgID != testUser.CurrentOrg.String() {
			t.Errorf("Expected org_id %s (from token), got %s", testUser.CurrentOrg.String(), deletionRequest.OrgID)
		}
	})
}
