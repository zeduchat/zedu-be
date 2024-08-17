package test_contact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
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

		currUUID := utility.GenerateUUID()
		contactData := models.ContactUs{
			Name:        "John Doe",
			Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			Message:     "I would like to know more about your services3.",
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
			Name:        "John Doe",
			Email:       "invalid_email",
			PhoneNumber: "8883344444",
			Message:     "message test",
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