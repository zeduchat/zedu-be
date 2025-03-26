package centrifuge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/centrifugal/gocent"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func NewCentrifugoService(logger *utility.Logger, config config.Centrifuge) *gocent.Client {

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	c := gocent.New(gocent.Config{
		Addr:       config.Url,
		Key:        config.ApiKey,
		HTTPClient: httpClient,
	})

	Client.C = c

	utility.LogAndPrint(logger, fmt.Sprintf("connected to centrifuge server at %s", config.Url))
	return c
}

func PublishChannel(logger *utility.Logger, channelID string, publishPayload interface{}) error {
	payload, err := json.Marshal(publishPayload)
	if err != nil {
		return err
	}

	client := Client.C

	err = client.Publish(context.Background(), channelID, payload)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to publish to channel %s: %v", channelID, err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("published to %s", channelID))

	return nil
}

func PublishToThreadSubChannel(logger *utility.Logger, channelID string, threadID string, publishPayload interface{}) error {

	subChannelID := fmt.Sprintf("%s:%s", channelID, threadID)
	payload, err := json.Marshal(publishPayload)
	if err != nil {
		return err
	}

	client := Client.C
	err = client.Publish(context.Background(), subChannelID, payload)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to publish to sub-channel %s: %v", subChannelID, err))
		return err
	}

	logger.Info(fmt.Sprintf("Published to sub-channel %s", subChannelID))

	return nil
}

func BatchBroadcastToChannel(logger *utility.Logger, channelIDs []string, publishPayload interface{}) error {
	ctx := context.Background()

	payload, err := json.Marshal(publishPayload)
	if err != nil {
		return err
	}

	client := Client.C

	if len(channelIDs) == 0 {
		return errors.New("no channels to broadcast to")
	}

	err = client.Broadcast(ctx, channelIDs, payload)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to broadcast to channel %s: %v", channelIDs, err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("broadcasted to %s", channelIDs))

	return nil
}
