package firebase

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func ConnectFirebase(logger *utility.Logger, config config.Firebase) {
	utility.LogAndPrint(logger, fmt.Sprint("Connect Firebase was called"))
	opt := option.WithCredentialsFile(config.ServiceFilePath)
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
	client := Connection().Client

	message := &messaging.Message{
		Token: fcmToken,
		Notification: &messaging.Notification{
			Title:    title,
			Body:     body,
			ImageURL: avatarUrl,
		},
	}

	// Send the message
	_, err := client.Send(context.Background(), message)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	utility.LogAndPrint(logger, "Successfully sent push notification")
	return nil
}

// Send push notification to multiple FCM tokens
func SendNotificationByFCMTokens(logger *utility.Logger, fcmTokens []string, title string, body string) error {
	client := Connection().Client

	notification := &messaging.Notification{
		Title: title,
		Body:  body,
	}

	response, err := client.SendEachForMulticast(context.Background(), &messaging.MulticastMessage{
		Tokens:       fcmTokens,
		Notification: notification,
	})

	if err != nil {
		return fmt.Errorf("error sending multicast message: %v", err)
	}

	utility.LogAndPrint(logger, fmt.Sprintf("Successfully sent message to %d recipients. Failure count: %d", response.SuccessCount, response.FailureCount))
	return nil
}
