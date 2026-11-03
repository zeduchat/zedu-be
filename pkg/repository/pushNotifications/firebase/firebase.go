package firebase

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

const fcmMulticastLimit = 500

func ConnectFirebase(logger *utility.Logger, config config.Firebase) {
	utility.LogAndPrint(logger, "Connect Firebase was called")
	serviceFilePath := strings.Trim(strings.TrimSpace(config.ServiceFilePath), "\"'")
	if serviceFilePath == "" {
		utility.LogAndPrint(logger, "FIREBASE_SERVICE_FILE_PATH is empty; Firebase push is disabled")
		return
	}
	data, err := os.ReadFile(serviceFilePath)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Firebase credentials file %q is not readable: %v", serviceFilePath, err))
		return
	}
	utility.LogAndPrint(logger, fmt.Sprintf("Firebase credentials file %s is readable (%d bytes)", serviceFilePath, len(data)))
	opt := option.WithCredentialsFile(serviceFilePath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("error initializing Firebase app: %v", err))
		return
	}

	// Get the messaging client from the Firebase app
	client, err := app.Messaging(context.Background())
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("error getting messaging client: %v", err))
		return
	}

	// Assign the app and client to the global Client variable
	Client.App = app
	Client.Client = client
	utility.LogAndPrint(logger, "Successfully initialized Firebase client")
}

// Send push notification to a single FCM token
func SendNotificationByFCMToken(logger *utility.Logger, fcmToken string, title string, body, avatarUrl string) error {
	_, _, err := SendPush(logger, []string{fcmToken}, title, body, avatarUrl, nil)
	return err
}

// Send push notification to multiple FCM tokens
func SendNotificationByFCMTokens(logger *utility.Logger, fcmTokens []string, title string, body string) error {
	_, _, err := SendPush(logger, fcmTokens, title, body, "", nil)
	return err
}

func SendPush(logger *utility.Logger, tokens []string, title, body, imageURL string, data map[string]string) ([]string, int, error) {
	client := Connection().Client
	if client == nil {
		if logger != nil {
			logger.Info("Firebase client is not initialized; skipping push delivery")
		}
		return nil, 0, nil
	}
	if len(tokens) == 0 {
		return nil, 0, nil
	}

	notification := &messaging.Notification{Title: title, Body: body}
	if imageURL != "" {
		notification.ImageURL = imageURL
	}

	apnsConfig := &messaging.APNSConfig{
		Headers: map[string]string{
			"apns-priority":  "10",
			"apns-push-type": "alert",
		},
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{Sound: "default"},
		},
	}
	if imageURL != "" {
		apnsConfig.FCMOptions = &messaging.APNSFCMOptions{ImageURL: imageURL}
	}

	webpushConfig := &messaging.WebpushConfig{
		Headers: map[string]string{
			"Urgency": "high",
		},
	}

	callData := data
	callMessage := false
	if isDirectCall(data) {
		callMessage = true
		callData = withDisplayData(data, title, body)
		apnsConfig.Payload.Aps.Alert = &messaging.ApsAlert{Title: title, Body: body}
		webpushConfig.Notification = &messaging.WebpushNotification{
			Title: title,
			Body:  body,
			Icon:  imageURL,
		}
	}

	var invalid []string
	successCount := 0
	realFailures := 0

	for start := 0; start < len(tokens); start += fcmMulticastLimit {
		end := start + fcmMulticastLimit
		if end > len(tokens) {
			end = len(tokens)
		}
		batch := tokens[start:end]

		message := &messaging.MulticastMessage{
			Tokens: batch,
			Data:   data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID: "zedu_messages",
					Sound:     "default",
				},
			},
			APNS:         apnsConfig,
			Webpush:      webpushConfig,
			Notification: notification,
		}
		if callMessage {
			// Data-only on Android so the app can show the incoming-call screen
			// while the app is backgrounded. Web and iOS still get a visible alert.
			message.Data = callData
			message.Notification = nil
			message.Android.Notification = nil
		}

		response, err := client.SendEachForMulticast(context.Background(), message)
		if err != nil {
			return invalid, successCount, fmt.Errorf("error sending multicast message: %v", err)
		}

		successCount += response.SuccessCount
		invalidArgTokens := make([]string, 0)
		for i, sendResponse := range response.Responses {
			if sendResponse.Success || sendResponse.Error == nil {
				continue
			}
			if messaging.IsUnregistered(sendResponse.Error) || messaging.IsSenderIDMismatch(sendResponse.Error) {
				invalid = append(invalid, batch[i])
				continue
			}
			realFailures++
			if messaging.IsInvalidArgument(sendResponse.Error) {
				invalidArgTokens = append(invalidArgTokens, batch[i])
			}
			logger.Error("firebase send failed for a token: %v", sendResponse.Error)
		}
		if response.SuccessCount > 0 {
			invalid = append(invalid, invalidArgTokens...)
		}
	}

	logger.Info("Sent Firebase push to %d tokens. Invalid tokens: %d", successCount, len(invalid))
	if successCount == 0 && realFailures > 0 {
		return invalid, successCount, fmt.Errorf("firebase delivered 0 of %d tokens", len(tokens))
	}
	return invalid, successCount, nil
}

func isDirectCall(data map[string]string) bool {
	if len(data) == 0 {
		return false
	}
	event := data["event"]
	kind := data["notification_type"]
	return strings.HasPrefix(event, "direct_call") || strings.HasPrefix(kind, "direct_call") || kind == "buzz_ended"
}

func withDisplayData(data map[string]string, title, body string) map[string]string {
	capHint := len(data)
	if capHint <= math.MaxInt-2 {
		capHint += 2
	}
	cloned := make(map[string]string, capHint)
	for key, value := range data {
		cloned[key] = value
	}
	if strings.TrimSpace(title) != "" {
		cloned["title"] = title
	}
	if strings.TrimSpace(body) != "" {
		cloned["body"] = body
	}
	return cloned
}
