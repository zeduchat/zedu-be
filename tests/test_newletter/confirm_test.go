package test_newsletter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func postConfirm(t *testing.T, body models.ConfirmNewsLetterRequest) *httptest.ResponseRecorder {
	t.Helper()
	router, _ := SetupNewsletterTestRouter()

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/newsletter/confirm", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	return resp
}

func TestConfirmNewsletter_ConfirmsPendingSubscriber(t *testing.T) {
	_, newsController := SetupNewsletterTestRouter()
	email := fmt.Sprintf("testuser%v@qa.team", utility.GenerateUUID())
	db := newsController.Db.Postgresql
	db.Create(&models.NewsLetter{Email: email, Pending: true})

	token := utility.NewsletterConfirmToken(email, config.GetConfig().Server.Secret)
	resp := postConfirm(t, models.ConfirmNewsLetterRequest{Email: email, Token: token})

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusOK)
	tst.AssertResponseMessage(t, response["message"].(string), "subscription confirmed successfully")

	var saved models.NewsLetter
	db.First(&saved, "email = ?", email)
	if saved.Pending {
		t.Error("the subscriber should be confirmed after a valid link is used")
	}
}

func TestConfirmNewsletter_RejectsBadToken(t *testing.T) {
	_, newsController := SetupNewsletterTestRouter()
	email := fmt.Sprintf("testuser%v@qa.team", utility.GenerateUUID())
	db := newsController.Db.Postgresql
	db.Create(&models.NewsLetter{Email: email, Pending: true})

	resp := postConfirm(t, models.ConfirmNewsLetterRequest{Email: email, Token: "not-a-valid-token"})

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	tst.AssertResponseMessage(t, response["message"].(string), "invalid or expired confirmation link")

	var saved models.NewsLetter
	db.First(&saved, "email = ?", email)
	if !saved.Pending {
		t.Error("the subscriber must stay pending when the token is wrong")
	}
}

func TestConfirmNewsletter_UnsubscribeTokenDoesNotConfirm(t *testing.T) {
	_, newsController := SetupNewsletterTestRouter()
	email := fmt.Sprintf("testuser%v@qa.team", utility.GenerateUUID())
	db := newsController.Db.Postgresql
	db.Create(&models.NewsLetter{Email: email, Pending: true})

	token := utility.NewsletterUnsubscribeToken(email, config.GetConfig().Server.Secret)
	resp := postConfirm(t, models.ConfirmNewsLetterRequest{Email: email, Token: token})

	tst.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
}

func TestConfirmNewsletter_UnknownEmailSucceeds(t *testing.T) {
	email := fmt.Sprintf("nobody%v@qa.team", utility.GenerateUUID())
	token := utility.NewsletterConfirmToken(email, config.GetConfig().Server.Secret)

	resp := postConfirm(t, models.ConfirmNewsLetterRequest{Email: email, Token: token})

	tst.AssertStatusCode(t, resp.Code, http.StatusOK)
}

func TestConfirmNewsletter_ValidatesBody(t *testing.T) {
	resp := postConfirm(t, models.ConfirmNewsLetterRequest{Email: "not-an-email", Token: ""})

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	tst.AssertResponseMessage(t, response["message"].(string), "Validation failed")
}
