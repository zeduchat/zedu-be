package onesignal

import (
	"context"
	"fmt"

	onesignalapi "github.com/OneSignal/onesignal-go-api/v5"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

// ConnectOneSignal initializes OneSignal client with provided configuration
func ConnectOneSignal(logger *utility.Logger, cfg config.OneSignal) {
	if !cfg.Enabled {
		utility.LogAndPrint(logger, "OneSignal is disabled in configuration")
		return
	}

	if cfg.AppID == "" || cfg.RestAPIKey == "" {
		utility.LogAndPrint(logger, "OneSignal AppID or RestAPIKey is empty, skipping initialization")
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

	utility.LogAndPrint(logger, "Successfully initialized OneSignal SDK v5 client")
}

// SendNotification sends a push notification to a single user via OneSignal subscription ID
func SendNotification(logger *utility.Logger, subscriptionID string, title string, body string) error {
	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		return fmt.Errorf("OneSignal client not initialized")
	}

	subscriptionIDs := []string{subscriptionID}
	return SendBatchNotifications(logger, subscriptionIDs, title, body)
}

// SendBatchNotifications sends a push notification to multiple users via OneSignal subscription IDs
func SendBatchNotifications(logger *utility.Logger, subscriptionIDs []string, title string, body string) error {
	if len(subscriptionIDs) == 0 {
		return fmt.Errorf("no subscription IDs provided")
	}

	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		return fmt.Errorf("OneSignal client not initialized")
	}

	// Create language string maps for multilingual support
	contentsMap := onesignalapi.LanguageStringMap{}
	contentsMap.SetEn(body)

	headingsMap := onesignalapi.LanguageStringMap{}
	headingsMap.SetEn(title)

	// Create notification using v5 SDK
	notification := onesignalapi.NewNotification(Client.AppID)
	notification.SetIncludeSubscriptionIds(subscriptionIDs)
	notification.SetContents(contentsMap)
	notification.SetHeadings(headingsMap)

	// Use context-based authentication for v5
	ctx := context.WithValue(context.Background(), onesignalapi.RestApiKey, Client.ApiKey)

	// Send notification via v5 SDK
	response, httpResp, err := Client.Client.DefaultApi.CreateNotification(ctx).
		Notification(*notification).
		Execute()

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Error sending OneSignal notification: %v", err))
		return fmt.Errorf("error sending OneSignal notification: %v", err)
	}

	if httpResp != nil && httpResp.StatusCode >= 400 {
		return fmt.Errorf("OneSignal API error (status %d)", httpResp.StatusCode)
	}

	if response == nil {
		return fmt.Errorf("OneSignal API returned nil response")
	}

	notificationID := ""
	if response.Id != nil {
		notificationID = *response.Id
	}

	utility.LogAndPrint(logger, fmt.Sprintf("Successfully sent OneSignal notification to %d recipients. ID: %s", len(subscriptionIDs), notificationID))
	return nil
}

// OptionalSendNotification sends a notification without failing if OneSignal is not initialized
func OptionalSendNotification(logger *utility.Logger, subscriptionID string, title string, body string) error {
	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		utility.LogAndPrint(logger, "OneSignal client not initialized, skipping notification")
		return nil
	}

	return SendNotification(logger, subscriptionID, title, body)
}

// OptionalSendBatchNotifications sends batch notifications without failing if OneSignal is not initialized
func OptionalSendBatchNotifications(logger *utility.Logger, subscriptionIDs []string, title string, body string) error {
	if Client.Client == nil || Client.AppID == "" || Client.ApiKey == "" {
		utility.LogAndPrint(logger, "OneSignal client not initialized, skipping batch notification")
		return nil
	}

	return SendBatchNotifications(logger, subscriptionIDs, title, body)
}
