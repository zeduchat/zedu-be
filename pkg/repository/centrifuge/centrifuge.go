package centrifuge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/centrifugal/centrifuge-go"
	"github.com/centrifugal/gocent"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func NewCentrifugoService(logger *utility.Logger, config config.Centrifuge) *gocent.Client {

	c := gocent.New(gocent.Config{
		Addr: "http://" + config.Url,
		Key:  config.ApiKey,
	})
	Client.C = c
	return c
}


func BroadcastChannel(logger *utility.Logger, channelID string, broadcastPayload interface{}) error {

	payload, err := json.Marshal(broadcastPayload)
	if err != nil {
		return err
	}

	client := Client.C

	err = client.Publish(context.Background(), channelID, payload)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to publish to channel %s: %v", channelID, err))
		return err
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("published to %s", channelID))

	return nil
}

func NewCentrifugod(logger *utility.Logger, config config.Centrifuge) *centrifuge.Client {
	centrifugoURL := fmt.Sprintf("ws://%s/connection/websocket", config.Url)
	fmt.Println(centrifugoURL)
	client := centrifuge.NewJsonClient(
		centrifugoURL,
		centrifuge.Config{
			GetToken: func(_ centrifuge.ConnectionTokenEvent) (string, error) {
				utility.LogAndPrint(logger, "Referesh connection event")
				token := GetConnToken()
				return token, nil
			},
		},
	)
	client.OnConnecting(func(e centrifuge.ConnectingEvent) {
		log.Printf("Connecting - %d (%s)", e.Code, e.Reason)
	})
	client.OnConnected(func(e centrifuge.ConnectedEvent) {
		log.Printf("Connected with ID %s", e.ClientID)
	})
	client.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
		log.Printf("Disconnected: %d (%s)", e.Code, e.Reason)
	})

	client.OnError(func(e centrifuge.ErrorEvent) {
		log.Printf("Error: %s", e.Error.Error())
	})

	client.OnMessage(func(e centrifuge.MessageEvent) {
		log.Printf("Message from server: %s", string(e.Data))
	})

	client.OnSubscribed(func(e centrifuge.ServerSubscribedEvent) {
		log.Printf("Subscribed to server-side channel %s: (was recovering: %v, recovered: %v)", e.Channel, e.WasRecovering, e.Recovered)
	})
	client.OnSubscribing(func(e centrifuge.ServerSubscribingEvent) {
		log.Printf("Subscribing to server-side channel %s", e.Channel)
	})
	client.OnUnsubscribed(func(e centrifuge.ServerUnsubscribedEvent) {
		log.Printf("Unsubscribed from server-side channel %s", e.Channel)
	})

	client.OnPublication(func(e centrifuge.ServerPublicationEvent) {
		log.Printf("Publication from server-side channel %s: %s (offset %d)", e.Channel, e.Data, e.Offset)
	})
	client.OnJoin(func(e centrifuge.ServerJoinEvent) {
		log.Printf("Join to server-side channel %s: %s (%s)", e.Channel, e.User, e.Client)
	})
	client.OnLeave(func(e centrifuge.ServerLeaveEvent) {
		log.Printf("Leave from server-side channel %s: %s (%s)", e.Channel, e.User, e.Client)
	})

	if err := client.Connect(); err != nil {
		utility.LogAndPrint(logger, "an error occured while connecting to centrifugo server: ", err)
		return nil
	}

	utility.LogAndPrint(logger, "connected to centrifugo server")

	sub, err := client.NewSubscription("channel", centrifuge.SubscriptionConfig{
		GetToken: func(e centrifuge.SubscriptionTokenEvent) (string, error) {
			utility.LogAndPrint(logger, "Referesh channel connection event")
			token := GetSubConnToken(e.Channel)
			return token, nil
		},
		Recoverable: true,
		JoinLeave:   true,
	})
	if err != nil {
		log.Fatalln(err)
	}

	sub.OnSubscribing(func(e centrifuge.SubscribingEvent) {
		log.Printf("Subscribing on channel %s - %d (%s)", sub.Channel, e.Code, e.Reason)
	})
	sub.OnSubscribed(func(e centrifuge.SubscribedEvent) {
		log.Printf("Subscribed on channel %s, (was recovering: %v, recovered: %v)", sub.Channel, e.WasRecovering, e.Recovered)
	})
	sub.OnUnsubscribed(func(e centrifuge.UnsubscribedEvent) {
		log.Printf("Unsubscribed from channel %s - %d (%s)", sub.Channel, e.Code, e.Reason)
	})

	sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		log.Printf("Subscription error %s: %s", sub.Channel, e.Error)
	})

	type ChatMessage struct {
		Input string `json:"input"`
	}

	sub.OnPublication(func(e centrifuge.PublicationEvent) {
		var chatMessage *ChatMessage
		err := json.Unmarshal(e.Data, &chatMessage)
		if err != nil {
			return
		}
		log.Printf("Someone says via channel %s: %s (offset %d)", sub.Channel, chatMessage.Input, e.Offset)
	})
	sub.OnJoin(func(e centrifuge.JoinEvent) {
		log.Printf("Someone joined %s: user id %s, client id %s", sub.Channel, e.User, e.Client)
	})
	sub.OnLeave(func(e centrifuge.LeaveEvent) {
		log.Printf("Someone left %s: user id %s, client id %s", sub.Channel, e.User, e.Client)
	})

	err = sub.Subscribe()
	if err != nil {
		log.Fatalln(err)
	}
	channelID := "public"
	broadcastPayload := map[string]interface{}{
		"something": "here",
	}

	payload, err := json.Marshal(broadcastPayload)
	if err != nil {
		panic(err)
	}

	pubRes, err := sub.Publish(context.Background(), payload)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to publish to channel %s: %v", channelID, err))
		panic(err)
	}

	utility.LogAndPrint(logger, fmt.Sprintf("published  %v to %s", pubRes, channelID))

	return nil
}

func GetConnToken() string {

	userClaims := jwt.MapClaims{"sub": "user1"}
	userClaims["exp"] = time.Now().Unix() + int64(30)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	connToken, err := token.SignedString([]byte(config.Config.Centrifuge.Secret))
	if err != nil {
		return connToken
	}

	return connToken
}

func GetSubConnToken(channelName string) string {

	userClaims := jwt.MapClaims{"sub": "user1"}
	userClaims["exp"] = time.Now().Unix() + int64(30)
	userClaims["channel"] = channelName

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	connToken, err := token.SignedString([]byte(config.Config.Centrifuge.Secret))
	if err != nil {
		return connToken
	}

	return connToken
}
