package rabbitmq

import (
	"encoding/json"
	"errors"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/rabbitmq"
	"gorm.io/gorm"
)

func PushToRabbitQueue(db *gorm.DB, feed models.FeedWebHookRequest) error {

	if rabbitmq.QueueClient.QueueManager == nil {
		return errors.New("rabbitmq client not initialized")
	}

	innerPayload := map[string]interface{}{
		"task": "telex_queue_processor.output_integrations",
		"args": []map[string]string{
			{
				"event_name": feed.EventName,
				"message":    feed.Content,
				"status":     feed.Status,
				"username":   feed.UserName,
			},
		},
	}

	innerPayloadBytes, err := json.Marshal(innerPayload)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"properties": map[string]interface{}{
			"content_type":  "application/json",
			"delivery_mode": 2,
		},
		"routing_key":      "telex_queue_processor.output_integrations",
		"payload":          string(innerPayloadBytes),
		"payload_encoding": "string",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Publish the message
	err = rabbitmq.QueueClient.QueueManager.Publish(
		[]string{"queue_name"},
		string(payloadBytes),
	)
	if err != nil {
		return err
	}

	return nil
}
