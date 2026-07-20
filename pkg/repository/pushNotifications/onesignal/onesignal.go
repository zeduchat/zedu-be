package onesignal

import (
	"context"
	"fmt"
	"strings"

	onesignalapi "github.com/OneSignal/onesignal-go-api/v5"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	onesignalService "github.com/hngprojects/telex_be/services/onesignal"
	"github.com/hngprojects/telex_be/utility"
)

// ConnectOneSignal initializes OneSignal client with provided configuration
func ConnectOneSignal(logger *utility.Logger, cfg config.OneSignal) {
	if !cfg.Enabled {
		logger.Info("OneSignal is disabled in configuration")
		return
	}

	if cfg.AppID == "" || cfg.RestAPIKey == "" {
		logger.Info("OneSignal AppID or RestAPIKey is empty, skipping initialization")
		return
	}

	// Create OneSignal API client configuration for v5
	apiConfig := onesignalapi.NewConfiguration()

	// Create the API client
	apiClient := onesignalapi.NewAPIClient(apiConfig)

	// Assign the client and credentials to the global Client variable
	Client.Client = apiClient
	Client.AppID = cfg.AppID
	Client.ApiKey = cfg.RestAPIKey

	logger.Info("Successfully initialized OneSignal SDK v5 client")
}

// SendNotification sends a push notification to a single user via OneSignal subscription ID
func SendNotification(logger *utility.Logger, subscriptionID string, req models.PushRequest, db *gorm.DB, userID string) error {
	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		return fmt.Errorf("OneSignal client not initialized")
	}

	subscriptionIDs := []string{subscriptionID}
	return SendBatchNotifications(logger, subscriptionIDs, req, db, []string{userID})
}

// SendBatchNotifications sends a push notification to multiple users via OneSignal subscription IDs
func SendBatchNotifications(logger *utility.Logger, subscriptionIDs []string, req models.PushRequest, db *gorm.DB, userIDs []string) error {
	if len(subscriptionIDs) == 0 {
		return fmt.Errorf("no subscription IDs provided")
	}

	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		return fmt.Errorf("OneSignal client not initialized")
	}

	// Create language string maps for multilingual support
	contentsMap := onesignalapi.LanguageStringMap{}
	contentsMap.SetEn(req.Message)

	headingsMap := onesignalapi.LanguageStringMap{}
	formattedTitle := fmt.Sprintf("%s", strings.Title(strings.ToLower(req.Title)))
	headingsMap.SetEn(formattedTitle)

	// Create notification using v5 SDK
	notification := onesignalapi.NewNotification(Client.AppID)
	notification.SetIncludeSubscriptionIds(subscriptionIDs)
	notification.SetContents(contentsMap)
	notification.SetHeadings(headingsMap)

	var payloadData map[string]interface{}
	if req.Payload != nil {
		if data, ok := req.Payload.(map[string]interface{}); ok {
			payloadData = data
			notification.SetData(data)
		}
	}

	if req.AvatarUrl != "" {
		notification.SetLargeIcon(req.AvatarUrl)
		notification.SetBigPicture(req.AvatarUrl)
	}

	// Use context-based authentication for v5
	ctx := context.WithValue(context.Background(), onesignalapi.RestApiKey, Client.ApiKey)

	// Send notification via v5 SDK
	response, httpResp, err := Client.Client.DefaultApi.CreateNotification(ctx).
		Notification(*notification).
		Execute()

	if err != nil {
		logger.Error("Error sending OneSignal notification: %v", err)
		return fmt.Errorf("error sending OneSignal notification: %v", err)
	}

	if httpResp != nil && httpResp.StatusCode >= 400 {
		logger.Error("OneSignal API error (status %d)", httpResp.StatusCode)
		return fmt.Errorf("OneSignal API error (status %d)", httpResp.StatusCode)
	}

	if response == nil {
		logger.Error("OneSignal API returned nil response")
		return fmt.Errorf("OneSignal API returned nil response")
	}

	notificationID := ""
	if response.Id != nil {
		notificationID = *response.Id
	}

	logger.Info("Successfully sent OneSignal notification to %d recipients. ID: %s", len(subscriptionIDs), notificationID)

	// Save notification to database if db and userIDs provided
	if db != nil && len(userIDs) > 0 && notificationID != "" {
		_, _, err := onesignalService.SaveBatchNotifications(db, notificationID, userIDs, req.Title, req.Message, req.AvatarUrl, payloadData, req.OrgId)
		if err != nil {
			logger.Error("Failed to save notification to database: %v", err)
			// Don't fail the notification flow if DB save fails
		} else {
			logger.Info("Successfully saved notification to database for %d users", len(userIDs))
		}
	}

	return nil
}

// OptionalSendNotification sends a notification without failing if OneSignal is not initialized
func OptionalSendNotification(logger *utility.Logger, subscriptionID string, req models.PushRequest, db *gorm.DB, userID string) error {
	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		logger.Info("OneSignal client not initialized, skipping notification")
		return nil
	}

	return SendNotification(logger, subscriptionID, req, db, userID)
}

// OptionalSendBatchNotifications sends batch notifications without failing if OneSignal is not initialized
func OptionalSendBatchNotifications(logger *utility.Logger, subscriptionIDs []string, req models.PushRequest, db *gorm.DB, userIDs []string) error {
	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		logger.Info("OneSignal client not initialized, skipping batch notification")
		return nil
	}

	return SendBatchNotifications(logger, subscriptionIDs, req, db, userIDs)
}

func SendDirectCallNotification(logger *utility.Logger, subscriptionIDs []string, req models.PushRequest, callData map[string]interface{}, db *gorm.DB, userIDs []string) error {
	if len(subscriptionIDs) == 0 {
		return fmt.Errorf("no subscription IDs provided")
	}

	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		return fmt.Errorf("OneSignal client not initialized")
	}

	contentsMap := onesignalapi.LanguageStringMap{}
	contentsMap.SetEn(req.Message)

	headingsMap := onesignalapi.LanguageStringMap{}
	formattedTitle := fmt.Sprintf("%s", strings.Title(strings.ToLower(req.Title)))
	headingsMap.SetEn(formattedTitle)

	notification := onesignalapi.NewNotification(Client.AppID)
	notification.SetIncludeSubscriptionIds(subscriptionIDs)
	notification.SetContents(contentsMap)
	notification.SetHeadings(headingsMap)
	notification.SetData(callData)

	if req.AvatarUrl != "" {
		notification.SetLargeIcon(req.AvatarUrl)
		notification.SetBigPicture(req.AvatarUrl)
	}

	ctx := context.WithValue(context.Background(), onesignalapi.RestApiKey, Client.ApiKey)

	response, httpResp, err := Client.Client.DefaultApi.CreateNotification(ctx).
		Notification(*notification).
		Execute()

	if err != nil {
		logger.Error("SendDirectCallNotification: error sending notification: %v", err)
		return fmt.Errorf("error sending direct call notification: %v", err)
	}

	if httpResp != nil && httpResp.StatusCode >= 400 {
		logger.Error("SendDirectCallNotification: API error (status %d)", httpResp.StatusCode)
		return fmt.Errorf("OneSignal API error (status %d)", httpResp.StatusCode)
	}

	if response == nil {
		logger.Error("OneSignal API returned nil response")
		return fmt.Errorf("OneSignal API returned nil response")
	}

	notificationID := ""
	if response.Id != nil {
		notificationID = *response.Id
	}

	logger.Info("Successfully sent OneSignal direct call notification to %d recipients. ID: %s", len(subscriptionIDs), notificationID)

	// Save notification to database if db and userIDs provided
	if db != nil && len(userIDs) > 0 && notificationID != "" {
		_, _, err := onesignalService.SaveBatchNotifications(db, notificationID, userIDs, req.Title, req.Message, req.AvatarUrl, callData, req.OrgId)
		if err != nil {
			logger.Error("Failed to save notification to database: %v", err)
			// Don't fail the notification flow if DB save fails
		} else {
			logger.Info("Successfully saved notification to database for %d users", len(userIDs))
		}
	}

	return nil
}
