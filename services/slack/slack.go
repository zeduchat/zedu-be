package slack

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func GetSlackChannels(db *gorm.DB, extReq request.ExternalRequest, userId string, organisationId string) ([]external_models.SlackChannel, error) {
	var slackTelex models.SlackTelex
	slackTelex.UserID = userId

	err := slackTelex.GetSlackAccessToken(db, userId, organisationId)
	if err != nil {
		return nil, err
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

func ExchangeSlackOAuthToken(db *gorm.DB, req models.OAuth, extReq request.ExternalRequest, userId string) (gin.H, error) {
	var slackTelex models.SlackTelex
	response, err := extReq.SendExternalRequest(request.SlackOAuthExchange, req.OauthCode)
	if err != nil {
		return nil, err
	}

	slackResponse, ok := response.(external_models.SlackOAuthResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	if slackResponse.Error != "" {
		return nil, fmt.Errorf("%v", slackResponse.Error)
	}

	var integration models.Integrations

	err = integration.GetIntegrationID(db, "Slack")

	if err != nil {
		return nil, err
	}

	slackTelex = models.SlackTelex{
		ID:               utility.GenerateUUID(),
		UserID:           userId,
		OrganisationID:   req.OrganisationID,
		IntegrationID:    integration.ID,
		AccessToken:      slackResponse.AccessToken,
		TeamID:           slackResponse.Team.ID,
		TeamName:         slackResponse.Team.Name,
		Channel:          slackResponse.IncomingWebHook.Channel,
		ChannelID:        slackResponse.IncomingWebHook.ChannelId,
		ConfigurationURL: slackResponse.IncomingWebHook.ConfigurationUrl,
		URL:              slackResponse.IncomingWebHook.Url,
	}

	err = slackTelex.Create(db)

	if err != nil {
		return nil, err
	}

	orgIntegration := models.OrganisationIntegrations{
		ID:            utility.GenerateUUID(),
		OrgID:         req.OrganisationID,
		IntegrationID: integration.ID,
		IsArchived:    false,
		IsActive:      true,
	}

	err = orgIntegration.CreateOrganisationIntegration(db)
	if err != nil {
		return nil, err
	}

	result := gin.H{
		"access_token":     slackResponse.AccessToken,
		"incoming_webhook": slackResponse.IncomingWebHook,
	}

	return result, nil
}

func GetSlackAccessToken(db *gorm.DB, userID, organizationID string, rds *redis.Client, extReq request.ExternalRequest) (models.SlackTelex, error) {
	var slackTelex models.SlackTelex
	slackTelex.UserID = userID

	if err := slackTelex.GetSlackAccessToken(db, userID, organizationID); err != nil {
		return models.SlackTelex{}, fmt.Errorf("could not find SlackTelex record: %v", err)
	}

	return slackTelex, nil
}
