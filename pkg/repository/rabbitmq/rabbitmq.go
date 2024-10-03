package rabbitmq

import (
    "fmt"
    "log"

    "github.com/rabbitmq/amqp091-go"
    "github.com/hngprojects/telex_be/internal/config"
    "github.com/hngprojects/telex_be/utility"
)


func NewQueueManager(logger *utility.Logger, config config.RabbitMQ) *QueueManager {
    conn, err := amqp091.Dial(config.Connection)
    if err != nil {
        utility.LogAndPrint(logger, fmt.Sprintf("failed to connect to RabbitMQ: %v", err))
        return nil
    }

    ch, err := conn.Channel()
    if err != nil {
        utility.LogAndPrint(logger, fmt.Sprintf("failed to open a channel: %v", err))
        conn.Close()
        return nil
    }

    QueueClient.QueueManager = &QueueManager{
        conn: conn,
        ch:   ch,
    }

    utility.LogAndPrint(logger, fmt.Sprintf("connected to RabbitMQ server at %s", config.Connection))

    return &QueueManager{
        conn: conn,
        ch:   ch,
    }
}

func (qm *QueueManager) Close() {
    qm.ch.Close()
    qm.conn.Close()
}

func (qm *QueueManager) Publish(queueNames []string, message string) error {
    for _, queueName := range queueNames {
        err := qm.ch.Publish(
            "",        // exchange
            queueName, // routing key
            false,     // mandatory
            false,     // immediate
            amqp091.Publishing{
                ContentType:  "text/plain",
                Body:         []byte(message),
                DeliveryMode: amqp091.Persistent,
            },
        )
        if err != nil {
            return fmt.Errorf("failed to publish a message: %v", err)
        }

        log.Printf(" [x] Sent %s", message)
    }

    return nil
}