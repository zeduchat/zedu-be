package slack

import (
	"fmt"
	"net/url"
	"strings"

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
	fmt.Println(params.Encode())

	err := r.getNewSendRequestObject(strings.NewReader(params.Encode()), headers, "/api/oauth.v2.access").SendRequest(&outBoundResponse)
	if err != nil {
		logger.Error("slack oauth", outBoundResponse, err.Error())
		return outBoundResponse, err
	}

	return outBoundResponse, nil
}

func (r *RequestObj) GetSlackChannels() (external_models.SlackChannelResponse, error) {
	var (
		outBoundResponse external_models.SlackChannelResponse
		logger           = r.Logger
		idata            = r.RequestData
	)

	accessToken, ok := idata.(string)
	if !ok {
		logger.Error("slack get channels", idata, "request data format error")
		return outBoundResponse, fmt.Errorf("request data format error")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	path := "/api/conversations.list"

	err := r.getNewSendRequestObject(nil, headers, path).SendRequest(&outBoundResponse)
	if err != nil {
		logger.Error("slack get channels", outBoundResponse, err.Error())
		return outBoundResponse, err
	}

	return outBoundResponse, nil
}

func (r *RequestObj) GetManifest(auth_token string) (external_models.SlackManifestResponse, error) {

	var (
		outBoundResponse external_models.SlackManifestResponse
		logger           = r.Logger
		idata            = r.RequestData
	)

	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "application/json",
		"Authorization": "Bearer " + auth_token,
	}

	path := fmt.Sprintf("?app_id=%s", config.Config.Slack.AppId)
	_ = idata

	err := r.getNewSendRequestObject(nil, headers, path).SendRequest(&outBoundResponse)
	if err != nil {
		logger.Error("slack get manifest", outBoundResponse, err.Error())
		return outBoundResponse, err
	}
	return outBoundResponse, nil
}

func (r *RequestObj) GetSlackToken(refresh_token string) (external_models.SlackTokenResponse, error) {
	var (
		outBoundResponse external_models.SlackTokenResponse
		logger           = r.Logger
		idata            = r.RequestData
	)

	path := fmt.Sprintf("/api/tooling.tokens.rotate?refresh_token=%s", refresh_token)
	_ = idata

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	err := r.getNewSendRequestObject(nil, headers, path).SendRequest(&outBoundResponse)
	if err != nil {
		logger.Error("slack get access token", outBoundResponse, err.Error())
		return outBoundResponse, err
	}

	return outBoundResponse, nil
}
