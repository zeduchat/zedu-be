package slack

import (
	"fmt"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
)

func ExchangeSlackOAuthToken(req models.SlackTelex, extReq request.ExternalRequest) (string, error) {

	response, err := extReq.SendExternalRequest(request.SlackOAuthExchange, req.OauthCode)
	if err != nil {
		return "", err
	}

	slackResponse, ok := response.(external_models.SlackOAuthResponse)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	if slackResponse.Error != "" {
		return "", fmt.Errorf("slack error: %v", slackResponse.Error)
	}

	return slackResponse.AccessToken, nil
}
