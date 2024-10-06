package rabbitmq

import (
	"fmt"

	"github.com/hngprojects/telex_be/pkg/repository/rabbitmq"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func PushToRabbitQueue(logger *utility.Logger, db *gorm.DB, payload, routing_key string) error {

	err := rabbitmq.QueueClient.QM.Publish(
		payload,
		routing_key,
	)
	if err != nil {
		return err
	}

	fmt.Println("Pushed to RabbitMQ queue successfully")
	logger.Info("Pushed to RabbitMQ queue successfully")

	return nil
}
