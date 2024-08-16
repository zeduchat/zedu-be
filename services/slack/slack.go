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
	var slackTelex models.SlackTelex
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

	slackTelex = models.SlackTelex{
		ID:             utility.GenerateUUID(),
		UserID:         userId,
		AccessToken:    slackResponse.AccessToken,
		OrganisationID: req.OrganisationID,
	}

	err = slackTelex.Create(db)

	if err != nil {
		return "", err
	}

	return slackResponse.AccessToken, nil
}

func GetSlackChannels(db *gorm.DB, extReq request.ExternalRequest, userId string, organisationId string) ([]external_models.SlackChannel, error) {
	var slackTelex models.SlackTelex
	slackTelex.UserID = userId

	err := slackTelex.GetSlackAccessToken(db, userId, organisationId)
	if err != nil {
		return nil, fmt.Errorf("could not find SlackTelex record: %v", err)
	}

	response, err := extReq.SendExternalRequest(request.SlackGetChannels, slackTelex.AccessToken)
	if err != nil {
		return nil, err
	}

	slackResponse, ok := response.(external_models.SlackChannelResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return slackResponse.Channels, nil
}
