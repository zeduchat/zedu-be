package rabbitmq

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQService struct {
	QM *QueueManager
}

var QueueClient *RabbitMQService = &RabbitMQService{}

type QueueManager struct {
	connection      *amqp091.Connection
	channel         *amqp091.Channel
	config          config.RabbitMQ
	done            chan bool
	mu              *sync.Mutex
	notifyConnClose chan *amqp091.Error
	notifyChanClose chan *amqp091.Error
	isReady         bool
	infoLog         *log.Logger
	errLog          *log.Logger
}

const (
	reconnectDelay        = 5 * time.Second
	reInitDelay           = 2 * time.Second
	resendDelay           = 5 * time.Second
	connectionIdleTimeout = 72 * time.Hour
)

var (
	errNotConnected = errors.New("not connected to a server")
	errShutdown     = errors.New("client is shutting down")
)
