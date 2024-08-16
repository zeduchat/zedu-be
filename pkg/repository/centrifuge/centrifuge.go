package centrifuge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/centrifugal/gocent"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func NewCentrifugoService(logger *utility.Logger, config config.Centrifuge) *gocent.Client {

	c := gocent.New(gocent.Config{
		Addr: config.Url,
		Key:  config.ApiKey,
	})
	Client.C = c

	utility.LogAndPrint(logger, fmt.Sprintf("connected to centrifuge server at %s", config.Url))
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

	utility.LogAndPrint(logger, fmt.Sprintf("published to %s", channelID))

	return nil
}
