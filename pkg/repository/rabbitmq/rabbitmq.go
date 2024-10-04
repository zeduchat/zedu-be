package rabbitmq

import (
	"fmt"
	"log"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
	"github.com/rabbitmq/amqp091-go"
)

func NewQueueManager() *QueueManager {
	return &QueueManager{
		conn: nil,
		ch:   nil,
	}
}

func (qm *QueueManager) Connect(logger *utility.Logger, config config.RabbitMQ) {
	conn, err := amqp091.DialConfig(
		config.Connection,
		amqp091.Config{
			Dial:      amqp091.DefaultDial(5 * time.Second),
			Heartbeat: 5 * time.Second,
		},
	)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to connect to RabbitMQ: %v", err))
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to open a channel: %v", err))
		conn.Close()
		return
	}

	QueueClient.QueueManager = &QueueManager{
		conn: conn,
		ch:   ch,
	}

	utility.LogAndPrint(logger, fmt.Sprintf("connected to RabbitMQ server at %s", config.Connection))
}

func (qm *QueueManager) Reconnect(logger *utility.Logger, config config.RabbitMQ, retries int, backoff time.Duration) {

	for i := 0; i < retries; i++ {
		utility.LogAndPrint(logger, fmt.Sprintf("Attempting to reconnect to RabbitMQ (%d/%d)...", i+1, retries))

		qm.Connect(logger, config)

		if QueueClient.QueueManager != nil && QueueClient.QueueManager.conn != nil {
			utility.LogAndPrint(logger, "Reconnection to RabbitMQ successful")
			return
		}

		utility.LogAndPrint(logger, fmt.Sprintf("Reconnection failed. Retrying in %s...", backoff))
		time.Sleep(backoff)

		// Exponentially increase the backoff time
		backoff *= 2
	}

	utility.LogAndPrint(logger, "Reconnection attempts finished.")

}

func (qm *QueueManager) DeclareQueue(config config.RabbitMQ, queue_name string) error {
	_, err := qm.ch.QueueDeclare(
		queue_name,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %v", err)
	}

	return nil
}

func (qm *QueueManager) Close() {
	qm.ch.Close()
	qm.conn.Close()
}

func (qm *QueueManager) Publish(logger *utility.Logger, config config.RabbitMQ, data, routing_key string) error {
	if qm.conn == nil || qm.conn.IsClosed() {
		utility.LogAndPrint(logger, "RabbitMQ connection is closed. Reconnecting...")
		qm.Reconnect(logger, config, 5, 2*time.Second)
	}

	err := qm.ch.Publish(
		config.Exchange, // exchange
		routing_key,     // routing key
		false,           // mandatory
		false,           // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         []byte(data),
			DeliveryMode: amqp091.Persistent,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish a message: %v", err)
	}

	log.Println(" [x][x][x] Sent Payload successfully [x][x][x]")

	return nil
}
