package slack

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

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
		return nil, fmt.Errorf("slack error: %v", slackResponse.Error)
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

func GetSlackAccessToken(db *gorm.DB, userId string, organisationId string) (models.SlackTelex, error) {
	var slackTelex models.SlackTelex
	slackTelex.UserID = userId

	err := slackTelex.GetSlackAccessToken(db, userId, organisationId)
	if err != nil {
		return models.SlackTelex{}, fmt.Errorf("could not find SlackTelex record: %v", err)
	}

	return slackTelex, nil
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

func GetManifest(db *gorm.DB, rds *redis.Client, extReq request.ExternalRequest, req models.SlackManifestRequest) (gin.H, error) {
	cachedKey := fmt.Sprintf("slack_manifest_%v", req.AppID)

	cachedManifest, err := rds.Get(rds.Context(), cachedKey).Result()
	if err == redis.Nil {

		response, err := extReq.SendExternalRequest(request.SlackGetManifest, req)
		if err != nil {
			return nil, err
		}

		fmt.Println(response)

		slackManifest, ok := response.(external_models.SlackManifestResponse)
		if !ok {
			return nil, fmt.Errorf("invalid response format")
		}


		slackManifestJSON, err := json.Marshal(slackManifest)
		if err != nil {
			return nil, fmt.Errorf("could not marshal slack manifest: %v", err)
		}

		err = rds.Set(rds.Context(), cachedKey, slackManifestJSON, 0).Err()
		if err != nil {
			return nil, fmt.Errorf("could not cache slack manifest: %v", err)
		}

		return gin.H{
			"manifest": slackManifest,
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not retrieve cached slack manifest: %v", err)
	}

	return gin.H{
		"manifest": cachedManifest,
	}, nil
}
