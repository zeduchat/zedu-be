package testoptin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestE2EOptInSubscription(t *testing.T) {
	router, _ := SetupOptInTestRouter()

	currUUID := utility.GenerateUUID()
	body := models.CreateOptIn{
		FirstName: "John",
		LastName:  "Doe",
		Email:     fmt.Sprintf("testuser%v@qa.team", currUUID),
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/api/v1/optin", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	tst.AssertStatusCode(t, resp.Code, http.StatusCreated)

	response := tst.ParseResponse(resp)
	tst.AssertResponseMessage(t, response["message"].(string), "Opted in successfully")
}

func TestPostOptIn_ValidateEmail(t *testing.T) {
	router, _ := SetupOptInTestRouter()

	currUUID := utility.GenerateUUID()
	body := models.CreateOptIn{
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     fmt.Sprintf("invalid_email%v", currUUID), // Invalid email
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/optin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	tst.AssertResponseMessage(t, response["message"].(string), "Invalid email address")
}

func TestPostOptIn_CheckDuplicateEmail(t *testing.T) {
	router, optInController := SetupOptInTestRouter()

	currUUID := utility.GenerateUUID()
	db := optInController.Db.Postgresql

	db.Create(&models.OptIn{
		ID:        utility.GenerateUUID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     fmt.Sprintf("testuser%v@qa.team", currUUID),
	})

	body := models.CreateOptIn{
		FirstName: "John",
		LastName:  "Doe",
		Email:     fmt.Sprintf("testuser%v@qa.team", currUUID),
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/optin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusConflict)
	tst.AssertResponseMessage(t, response["message"].(string), "email already opted in, please use a different email or stay tuned")
}

func TestPostOptIn_SaveData(t *testing.T) {
	router, optInController := SetupOptInTestRouter()

	currUUID := utility.GenerateUUID()
	body := models.CreateOptIn{
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     fmt.Sprintf("testuser%v@qa.team", currUUID),
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/optin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusCreated)
	tst.AssertResponseMessage(t, response["message"].(string), "Opted in successfully")

	var optIn models.OptIn
	optInController.Db.Postgresql.First(&optIn, "email = ?", fmt.Sprintf("testuser%v@qa.team", currUUID))
	if optIn.Email != fmt.Sprintf("testuser%v@qa.team", currUUID) {
		t.Errorf("Data not saved correctly to the database: expected email %s, got %s", fmt.Sprintf("testuser%v@qa.team", currUUID), optIn.Email)
	}
}

func TestPostOptIn_ResponseAndStatusCode(t *testing.T) {
	router, _ := SetupOptInTestRouter()

	currUUID := utility.GenerateUUID()
	body := models.CreateOptIn{
		FirstName: "John",
		LastName:  "Doe",
		Email:     fmt.Sprintf("testuser%v@gmail.com", currUUID),
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/optin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusCreated)
	tst.AssertResponseMessage(t, response["message"].(string), "Opted in successfully")
}
