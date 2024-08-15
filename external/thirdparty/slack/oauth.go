package slack

import (
	"net/url"
	"strings"
	"fmt"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/internal/config"
)

func (r *RequestObj) ExchangeSlackOAuthToken() (external_models.SlackOAuthResponse, error) {
	var (
		config           = config.GetConfig()
		outBoundResponse external_models.SlackOAuthResponse
		logger           = r.Logger
		idata            = r.RequestData
	)

	code, ok := idata.(string)
	if !ok {
		logger.Error("slack oauth", idata, "request data format error")
		return outBoundResponse, fmt.Errorf("request data format error")
	}

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	params := url.Values{}
	params.Add("client_id", config.Slack.ClientId)
	params.Add("client_secret", config.Slack.ClientSecret)
	params.Add("code", code)
	params.Add("redirect_uri", config.Slack.RedirectURI)

	fmt.Println("THIS CODE", params.Encode())

	logger.Info("slack oauth", code)
	err := r.getNewSendRequestObject(strings.NewReader(params.Encode()), headers, "/api/oauth.v2.access").SendRequest(&outBoundResponse)
	if err != nil {
		logger.Error("slack oauth", outBoundResponse, err.Error())
		return outBoundResponse, err
	}

	return outBoundResponse, nil
}