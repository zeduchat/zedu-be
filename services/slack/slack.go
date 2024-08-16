package slack

import (
	"fmt"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func ExchangeSlackOAuthToken(db *gorm.DB, req models.OAuth, extReq request.ExternalRequest, userId string) (string, error) {

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

	var slackTelex models.SlackTelex

	slackTelex = models.SlackTelex{
		ID:          utility.GenerateUUID(),
		UserID:      userId,
		AccessToken: slackResponse.AccessToken,
	}

	err = slackTelex.Create(db)

	return slackResponse.AccessToken, nil
}
