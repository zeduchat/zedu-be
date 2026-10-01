package centrifuge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/centrifugal/gocent"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)
type PublishEntry struct {
	Channel string
	Payload any
}

const (
	maxPipeBatchSize     = 250
	maxConcurrentBatches = 20
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

func PublishChannel(logger *utility.Logger, channelID string, publishPayload any) error {
	payload, err := json.Marshal(publishPayload)
	if err != nil {
		return err
	}

	client := Client.C

	if channelID == "" {
		if logger != nil {
			logger.Error("published to %s failed, empty channel_id supplied", channelID)
		}
		return fmt.Errorf("empty channel_id supplied")
	}

	err = client.Publish(context.Background(), channelID, payload)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to publish to channel %s: %v", channelID, err.Error()))
		return err
	}

	if logger != nil {
		logger.Info(fmt.Sprintf("published to %s", channelID))
	}
	return nil
}

func PublishChannelOptional(logger *utility.Logger, channelID string, publishPayload any) error {
	if Client == nil || Client.C == nil {
		if logger != nil {
			logger.Warning("Centrifuge client not initialized, skipping publish to channel %s", channelID)
		}
		return nil
	}
	return PublishChannel(logger, channelID, publishPayload)
}

func PublishToThreadSubChannel(logger *utility.Logger, channelID string, threadID string, publishPayload any) error {

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

func BatchBroadcastToChannel(logger *utility.Logger, channelIDs []string, publishPayload any) error {
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

	logger.Info(fmt.Sprintf("broadcasted to %d channels", len(channelIDs)))

	return nil
}

func PublishLeaveBuzzEvent(logger *utility.Logger, channelID string, publishPayload any) error {

	payload, err := json.Marshal(publishPayload)
	if err != nil {
		return err
	}

	client := Client.C
	err = client.Publish(context.Background(), channelID, payload)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to publish to channelID %s: %v", channelID, err))
		return err
	}

	logger.Info(fmt.Sprintf("Published leave event to buzz-channel %s", channelID))
	return nil

}


func PipePublishChannels(logger *utility.Logger, entries []PublishEntry) {
	client := Client.C

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentBatches)

	for i := 0; i < len(entries); i += maxPipeBatchSize {
		end := i + maxPipeBatchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]

		wg.Add(1)
		sem <- struct{}{}
		go func(batch []PublishEntry) {
			defer wg.Done()
			defer func() { <-sem }()

			pipe := client.Pipe()
			for _, e := range batch {
				data, err := json.Marshal(e.Payload)
				if err != nil {
					logger.Error("Failed to marshal payload for channel %s: %v", e.Channel, err)
					continue
				}
				pipe.AddPublish(e.Channel, data)
			}

			replies, err := client.SendPipe(context.Background(), pipe)
			if err != nil {
				logger.Error("Pipe send failed: %v", err)
				for j, reply := range replies {
					if reply.Error != nil {
						logger.Error("Publish failed for channel %s: %v", batch[j].Channel, reply.Error)
					}
				}
			}
		}(batch)
	}

	wg.Wait()
}

