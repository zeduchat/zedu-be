package webpush

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

// PushClient is a reusable VAPID-enabled web push client
type PushClient struct {
	vapidPublicKey  string
	vapidPrivateKey string
}

func NewPushClient(logger *utility.Logger, webPush config.WebPush) *PushClient {
	Client.C = &PushClient{
		vapidPublicKey:  webPush.PubKey,
		vapidPrivateKey: webPush.PrivKey,
	}

	logger.Info("initialized webpush succesfully")

	return Client.C
}

// SendPush sends a push notification to a given subscription
func SendPush(subscription webpush.Subscription, payload any) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := webpush.SendNotification(message, &subscription, &webpush.Options{
		Subscriber:      "mailto:alayosingers@gmail.com",
		VAPIDPublicKey:  Client.C.vapidPublicKey,
		VAPIDPrivateKey: Client.C.vapidPrivateKey,
		TTL:             30,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	})
	if err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return nil
}
