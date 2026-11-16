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

func postUnsubscribe(t *testing.T, body models.UnsubscribeNewsLetterRequest) *httptest.ResponseRecorder {
	t.Helper()
	router, _ := SetupNewsletterTestRouter()

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/newsletter/unsubscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	return resp
}

func TestUnsubscribeNewsletter_RemovesSubscriber(t *testing.T) {
	_, newsController := SetupNewsletterTestRouter()
	email := fmt.Sprintf("testuser%v@qa.team", utility.GenerateUUID())
	db := newsController.Db.Postgresql
	db.Create(&models.NewsLetter{Email: email})

	token := utility.NewsletterUnsubscribeToken(email, config.GetConfig().Server.Secret)
	resp := postUnsubscribe(t, models.UnsubscribeNewsLetterRequest{Email: email, Token: token})

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusOK)
	tst.AssertResponseMessage(t, response["message"].(string), "email unsubscribed successfully")

	var count int64
	db.Unscoped().Model(&models.NewsLetter{}).Where("email = ?", email).Count(&count)
	if count != 0 {
		t.Errorf("expected the subscriber to be removed, found %d rows", count)
	}
}

func TestUnsubscribeNewsletter_RejectsBadToken(t *testing.T) {
	_, newsController := SetupNewsletterTestRouter()
	email := fmt.Sprintf("testuser%v@qa.team", utility.GenerateUUID())
	db := newsController.Db.Postgresql
	db.Create(&models.NewsLetter{Email: email})

	resp := postUnsubscribe(t, models.UnsubscribeNewsLetterRequest{Email: email, Token: "not-a-valid-token"})

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	tst.AssertResponseMessage(t, response["message"].(string), "invalid or expired unsubscribe link")

	var count int64
	db.Model(&models.NewsLetter{}).Where("email = ?", email).Count(&count)
	if count != 1 {
		t.Errorf("subscriber must stay on the list when the token is wrong, found %d rows", count)
	}
}

func TestUnsubscribeNewsletter_UnknownEmailSucceeds(t *testing.T) {
	email := fmt.Sprintf("nobody%v@qa.team", utility.GenerateUUID())
	token := utility.NewsletterUnsubscribeToken(email, config.GetConfig().Server.Secret)

	resp := postUnsubscribe(t, models.UnsubscribeNewsLetterRequest{Email: email, Token: token})

	tst.AssertStatusCode(t, resp.Code, http.StatusOK)
}

func TestUnsubscribeNewsletter_ValidatesBody(t *testing.T) {
	resp := postUnsubscribe(t, models.UnsubscribeNewsLetterRequest{Email: "not-an-email", Token: ""})

	response := tst.ParseResponse(resp)
	tst.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	tst.AssertResponseMessage(t, response["message"].(string), "Validation failed")
}

func TestUnsubscribeNewsletter_CanResubscribeAfterwards(t *testing.T) {
	router, newsController := SetupNewsletterTestRouter()
	email := fmt.Sprintf("testuser%v@qa.team", utility.GenerateUUID())
	newsController.Db.Postgresql.Create(&models.NewsLetter{Email: email})

	token := utility.NewsletterUnsubscribeToken(email, config.GetConfig().Server.Secret)
	postUnsubscribe(t, models.UnsubscribeNewsLetterRequest{Email: email, Token: token})

	jsonBody, _ := json.Marshal(models.NewsLetter{Email: email})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/newsletter", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	tst.AssertStatusCode(t, resp.Code, http.StatusCreated)
}
