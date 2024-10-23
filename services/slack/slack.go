package slack

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
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

	fmt.Println(slackResponse, response)

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



func getManifest(db *gorm.DB, rds *redis.Client, extReq request.ExternalRequest, token string) (external_models.SlackManifestResponse, error) {
	response, err := extReq.SendExternalRequest(request.SlackGetManifest, token)
	if err != nil {
		return nil, err
	}

	slackManifest, ok := response.(external_models.SlackManifestResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return slackManifest, nil
}

func GetSlackAccessToken(db *gorm.DB, userID, organizationID string, rds *redis.Client, extReq request.ExternalRequest) (models.SlackTelex, error) {
	var slackTelex models.SlackTelex
	slackTelex.UserID = userID

	// access_token := config.Config.Slack.RefreshToken

	// manifestResponse, err := getManifest(db, rds, extReq, access_token)
	// if err != nil {
	// 	return models.SlackTelex{}, fmt.Errorf("could not retrieve slack manifest: %v", err)
	// }
	
	if err := slackTelex.GetSlackAccessToken(db, userID, organizationID); err != nil {
		return models.SlackTelex{}, fmt.Errorf("could not find SlackTelex record: %v", err)
	}

	// slackTelex.AppManifest = models.JSONB(manifestResponse)
	
	return slackTelex, nil
}

// func GetSlackAccessToken(db *gorm.DB, userID, organizationID string, rds *redis.Client, extReq request.ExternalRequest) (models.SlackTelex, error) {
// 	var slackTelex models.SlackTelex
// 	slackTelex.UserID = userID

// 	token, err := getOrCreateSlackToken(db, userID, organizationID, extReq)
// 	if err != nil {
// 		return models.SlackTelex{}, fmt.Errorf("error getting or creating slack token: %v", err)
// 	}

// 	if time.Now().After(token.ExpiryTime) {
// 		if err := refreshToken(db, extReq, userID, organizationID, token); err != nil {
// 			return models.SlackTelex{}, fmt.Errorf("error refreshing token: %v", err)
// 		}
// 	}


// 	manifestResponse, err := GetManifest(db, rds, extReq, token.AccessToken)
// 	if err != nil {
// 		return models.SlackTelex{}, fmt.Errorf("could not retrieve slack manifest: %v", err)
// 	}

// 	slackTelex.AppManifest = models.JSONB(manifestResponse)

// 	if err := slackTelex.GetSlackAccessToken(db, userID, organizationID); err != nil {
// 		return models.SlackTelex{}, fmt.Errorf("could not find SlackTelex record: %v", err)
// 	}

// 	return slackTelex, nil
// }

func getOrCreateSlackToken(db *gorm.DB, userID, organizationID string, extReq request.ExternalRequest) (*models.SlackToken, error) {
	var (
		token         models.SlackToken
		refresh_token string
	)

	result := db.Where("user_id = ? AND organisation_id = ?", userID, organizationID).First(&token)

	if result.Error == gorm.ErrRecordNotFound {
		// Create a new token entry with just the refresh token
		refresh_token = config.Config.Slack.RefreshToken

		token = models.SlackToken{
			ID:             utility.GenerateUUID(),
			UserID:         userID,
			OrganisationID: organizationID,
			RefreshToken:   refresh_token,
			ExpiryTime:     time.Now(), // Set to current time to force immediate refresh
		}

		err := postgresql.CreateOneRecord(db, &token)
		if err != nil {
			return nil, fmt.Errorf("error creating new token: %v", err)
		}
	} else if result.Error != nil {
		return nil, fmt.Errorf("error querying database: %v", result.Error)
	}

	// Check if access token exists, if not, generate it
	if token.AccessToken == "" {
		if err := refreshToken(db, extReq, userID, organizationID, &token); err != nil {
			return nil, fmt.Errorf("error generating initial access token: %v", err)
		}
	}

	return &token, nil
}

func refreshToken(db *gorm.DB, extReq request.ExternalRequest, userID, orgID string, token *models.SlackToken) error {
	response, err := extReq.SendExternalRequest(request.SlackGetAccessToken, token.RefreshToken)
	if err != nil {
		return fmt.Errorf("error refreshing token: %v", err)
	}

	slackResponse, ok := response.(external_models.SlackTokenOutput)
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	token.AccessToken = slackResponse.Token
	token.RefreshToken = slackResponse.RefreshToken
	token.ExpiryTime = time.Now().Add(11 * time.Hour)

	err = token.UpdateToken(db, userID, orgID, slackResponse.Token, slackResponse.RefreshToken)
	if err != nil {
		return fmt.Errorf("error updating token: %v", err)
	}

	return nil
}

// func GetSlackAccessToken(db *gorm.DB, userId string, organisationId string, rds *redis.Client, extReq request.ExternalRequest) (models.SlackTelex, error) {
// 	var (
// 		slackTelex    models.SlackTelex
// 		slackToken    models.SlackToken
// 		tokens        models.SlackToken
// 		access_token  string
// 	)
// 	slackTelex.UserID = userId

// 	IsEmpty := slackToken.IsEmpty(db)
// 	if IsEmpty {
// 		//fetch refresh_token from config
// 		tokens.RefreshToken = config.Config.Slack.RefreshToken
// 	} else {
// 		//fetch refresh_token from db
// 		fetchedTokens, err := slackToken.GetToken(db, userId, organisationId)
// 		if err != nil {
// 			return models.SlackTelex{}, fmt.Errorf("could not retrieve stored slack access token: %v", err)
// 		}

// 		tokens = fetchedTokens
// 	}

// 	access_token = tokens.AccessToken

// 	//check if accesstoken has expired
// 	if time.Now().After(tokens.ExpiryTime) {
// 		//if expired, geneerate new accesstoken and update the db with the new accesstoken and refresh token

// 		tokenResponse, err := GenerateManifestTokens(db, extReq, tokens.RefreshToken)
// 		if err != nil {
// 			return models.SlackTelex{}, fmt.Errorf("could not retrieve slack access token: %v", err)
// 		}

// 		access_token = tokenResponse.AccessToken

// 		//update the db with the new accesstoken and refresh token
// 		err = slackToken.UpdateToken(db, userId, organisationId, tokenResponse.AccessToken, tokenResponse.RefreshToken)
// 		if err != nil {
// 			return models.SlackTelex{}, fmt.Errorf("could not update slack access token: %v", err)
// 		}

// 	}

// 	fmt.Println("TOKEN=============", access_token)

// 	manifestResponse, err := GetManifest(db, rds, extReq, access_token)
// 	if err != nil {
// 		return models.SlackTelex{}, fmt.Errorf("could not retrieve slack manifest: %v", err)
// 	}

// 	fmt.Println("MANIFESTRESPONSE=====================", manifestResponse)

// 	slackTelex.AppManifest = models.JSONB(manifestResponse)

// 	err = slackTelex.GetSlackAccessToken(db, userId, organisationId)
// 	if err != nil {
// 		return models.SlackTelex{}, fmt.Errorf("could not find SlackTelex record: %v", err)
// 	}

// 	return slackTelex, nil
// }

// func GenerateManifestTokens(db *gorm.DB, extReq request.ExternalRequest, refToken string) (external_models.SlackTokenResponse, error) {
// 	var (
// 		slackToken    models.SlackToken
// 		refresh_token string
// 	)
// 	//fetch refresh_token from config if the slacktoken table is empty
// 	isEmpty := slackToken.IsEmpty(db)
// 	if isEmpty {
// 		//fetch refresh_token from config
// 		refresh_token = config.Config.Slack.RefreshToken

// 		//store the refresh_token in the db
// 		err := slackToken.Create(db, refresh_token)
// 		if err != nil {
// 			return external_models.SlackTokenResponse{}, fmt.Errorf("could not store slack refresh token: %v", err)
// 		}

// 	} else {
// 		refresh_token = refToken
// 	}

// 	response, err := extReq.SendExternalRequest(request.SlackGetAccessToken, refresh_token)
// 	if err != nil {
// 		return external_models.SlackTokenResponse{}, err
// 	}

// 	slackResponse, ok := response.(external_models.SlackTokenResponse)
// 	if !ok {
// 		return external_models.SlackTokenResponse{}, fmt.Errorf("invalid response format")
// 	}

// 	return slackResponse, nil
// }
