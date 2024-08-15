package test_contact

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
)

func TestAddToContactUs(t *testing.T) {
	_, contactController := SetupContactTestRouter()
	db := contactController.Db.Postgresql

	setup := func() *gin.Engine {
		router, _ := SetupContactTestRouter()

		return router
	}

	t.Run("Successful Create Contact Us", func(t *testing.T) {
		router := setup()

		contactData := models.ContactUs{
			Name:    "John Doe",
			Email:   "johndoe@example.com",
			Message: "I would like to know more about your services3.",
		}
		payload, _ := json.Marshal(contactData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/contact", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Message sent successfully")

		var createdContact models.ContactUs
		db.Last(&createdContact)
		if createdContact.Email != contactData.Email {
			t.Errorf("Expected contact email %s, but got %s", contactData.Email, createdContact.Email)
		}
	})

	t.Run("Missing Fields - Bad Request", func(t *testing.T) {
		router := setup()

		contactData := models.ContactUs{
			Name: "John Doe",
		}
		payload, _ := json.Marshal(contactData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/contact", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})

	t.Run("Invalid Field Values - Unprocessable Entity", func(t *testing.T) {
		router := setup()

		contactData := models.ContactUs{
			Name:    "John Doe",
			Email:   "invalid_email",
			Message: "message test",
		}
		payload, _ := json.Marshal(contactData)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/contact", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})

}
