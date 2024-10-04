package rabbitmq

import "github.com/rabbitmq/amqp091-go"

type RabbitMQClient struct {
	QueueManager *QueueManager
}

var QueueClient *RabbitMQClient = &RabbitMQClient{}

type QueueManager struct {
	conn *amqp091.Connection
	ch   *amqp091.Channel
}

// func Connection() *RabbitMQClient {
// 	return QueueClient
// }