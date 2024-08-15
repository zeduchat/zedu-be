package slack

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
)

func ExchangeSlackOAuthToken(reqBody models.SlackTelex) (string, error) {
	var (
		config = config.GetConfig()
	)

	form := url.Values{}
	form.Add("client_id", config.Slack.ClientId)
	form.Add("client_secret", config.Slack.ClientSecret)
	form.Add("code", reqBody.OauthCode)
	form.Add("redirect_uri", config.Slack.RedirectURI)

	req, err := http.NewRequest("POST", "https://slack.com/api/oauth.v2.access", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var responseBody struct {
		AccessToken string `json:"access_token"`
		Ok          bool   `json:"ok"`
		Error       string `json:"error"`
	}

	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		return "", err
	}

	if !responseBody.Ok {
		return "", errors.New(responseBody.Error)
	}

	return responseBody.AccessToken, nil
}
