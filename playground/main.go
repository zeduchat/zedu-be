package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQService manages RabbitMQ connection and publishing.
type RabbitMQService struct {
	mu              *sync.Mutex
	queueName       string
	infolog         *log.Logger
	errlog          *log.Logger
	connection      *amqp.Connection
	channel         *amqp.Channel
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	isReady         bool
	done            chan bool
}

const (
	reconnectDelay        = 5 * time.Second
	reInitDelay           = 2 * time.Second
	connectionIdleTimeout = 10 * time.Second
)

var (
	errNotConnected = errors.New("not connected to the server")
	errShutdown     = errors.New("client is shutting down")
)

// NewRabbitMQService creates a new RabbitMQService instance.
func NewRabbitMQService(queueName, addr string) *RabbitMQService {
	return &RabbitMQService{
		mu:        &sync.Mutex{},
		infolog:   log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lmsgprefix),
		errlog:    log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmsgprefix),
		queueName: queueName,
		done:      make(chan bool),
	}
}

// Start initializes the RabbitMQ connection and sets up reconnection logic.
func (svc *RabbitMQService) Start(addr string) {
	go svc.handleReconnect(addr)
}

// handleReconnect manages the reconnection logic.
func (svc *RabbitMQService) handleReconnect(addr string) {
	for {
		svc.mu.Lock()
		svc.isReady = false
		svc.mu.Unlock()

		svc.infolog.Println("Attempting to connect to RabbitMQ...")

		conn, err := svc.connect(addr)
		if err != nil {
			svc.errlog.Println("Failed to connect. Retrying...")

			select {
			case <-svc.done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		if done := svc.handleReInit(conn); done {
			break
		}
	}
}

// connect establishes a new AMQP connection.
func (svc *RabbitMQService) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}

	svc.changeConnection(conn)
	svc.infolog.Println("Connected to RabbitMQ.")
	return conn, nil
}

// handleReInit will wait for a channel error and continuously attempt to reinitialize.
func (svc *RabbitMQService) handleReInit(conn *amqp.Connection) bool {
	for {
		svc.mu.Lock()
		svc.isReady = false
		svc.mu.Unlock()

		err := svc.init(conn)
		if err != nil {
			svc.errlog.Println("Failed to initialize channel. Retrying...")

			select {
			case <-svc.done:
				return true
			case <-svc.notifyConnClose:
				svc.infolog.Println("Connection closed, reconnecting...")
				return false
			case <-time.After(reInitDelay):
			}
			continue
		}

		select {
		case <-svc.done:
			return true
		case <-svc.notifyConnClose:
			svc.infolog.Println("Connection closed, reconnecting...")
			return false
		case <-svc.notifyChanClose:
			svc.infolog.Println("Channel closed, re-running init...")
		}
	}
}

// init initializes the RabbitMQ channel and declares the queue.
func (svc *RabbitMQService) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	err = ch.Confirm(false)
	if err != nil {
		return err
	}
	_, err = ch.QueueDeclare(
		svc.queueName,
		false, // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return err
	}

	svc.changeChannel(ch)
	svc.mu.Lock()
	svc.isReady = true
	svc.mu.Unlock()
	svc.infolog.Println("Channel and queue initialization done.")
	return nil
}

// changeConnection updates the RabbitMQ connection state.
func (svc *RabbitMQService) changeConnection(conn *amqp.Connection) {
	svc.connection = conn
	svc.notifyConnClose = make(chan *amqp.Error, 1)
	svc.connection.NotifyClose(svc.notifyConnClose)
}

// changeChannel updates the RabbitMQ channel state.
func (svc *RabbitMQService) changeChannel(channel *amqp.Channel) {
	svc.channel = channel
	svc.notifyChanClose = make(chan *amqp.Error, 1)
	svc.channel.NotifyClose(svc.notifyChanClose)
}

// Publish sends a message to the RabbitMQ queue.
func (svc *RabbitMQService) Publish(data []byte) error {
	svc.mu.Lock()
	if !svc.isReady {
		svc.mu.Unlock()
		return errNotConnected
	}
	svc.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return svc.channel.PublishWithContext(
		ctx,
		"",            // Exchange
		svc.queueName, // Routing key
		false,         // Mandatory
		false,         // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
}

// Close gracefully shuts down the RabbitMQ connection.
func (svc *RabbitMQService) Close() error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if !svc.isReady {
		return errShutdown
	}
	close(svc.done)
	err := svc.channel.Close()
	if err != nil {
		return err
	}
	err = svc.connection.Close()
	if err != nil {
		return err
	}

	svc.isReady = false
	return nil
}

func main() {
	queueName := "telex_queue_processor.filter_integrations"
	addr := "amqp://telexadmin:lovelybonds@49.12.208.6:5672/telexvhost"

	// Initialize and start the RabbitMQ service
	rabbitService := NewRabbitMQService(queueName, addr)
	fmt.Println(rabbitService)
	rabbitService.Start(addr)

	// Create a Gin router
	router := gin.Default()

	// Define the publish endpoint
	router.POST("/publish", func(c *gin.Context) {
		message := []byte("message from shallom") // Example message to be published
		err := rabbitService.Publish(message)
		if err != nil {
			log.Printf("Failed to publish message: %s\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to publish message",
				"error":   err.Error(),
			})
			return
		}

		log.Println("Message published successfully")
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Message published successfully",
		})
	})

	// Start the Gin server
	err := router.Run(":8080") // Run the server on port 8080
	if err != nil {
		log.Fatalf("Failed to start Gin server: %s", err)
	}
}
