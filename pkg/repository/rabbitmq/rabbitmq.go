package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
	"github.com/rabbitmq/amqp091-go"
)

func NewQueueManager(config config.RabbitMQ) *QueueManager {
	return &QueueManager{
		mu:      &sync.Mutex{},
		done:    make(chan bool),
		config:  config,
		infoLog: log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lmsgprefix),
		errLog:  log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (qm *QueueManager) Start(logger *utility.Logger) {
	go qm.HandleReconnect( qm.config.Connection, logger)
}

func (qm *QueueManager) HandleReconnect(addr string, logger *utility.Logger) {
	for {
		qm.mu.Lock()
		qm.isReady = false
		qm.mu.Unlock()

		logger.Info("Attempting to connect to RabbitMQ server...")
		qm.infoLog.Println("Attempting to connect to RabbitMQ...")

		conn, err := qm.Connect(addr, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to connect to RabbitMQ server: %v. Retrying...", err))
			qm.errLog.Printf("Failed to connect to RabbitMQ server: %v. Retrying...", err)

			select {
			case <-qm.done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		if done := qm.HandleReInit(conn, logger); done {
			break
		}
	}
}

func (qm *QueueManager) Connect(addr string, logger *utility.Logger) (*amqp091.Connection, error) {
	conn, err := amqp091.DialConfig(
		addr,
		amqp091.Config{
			Dial:      amqp091.DefaultDial(5 * time.Second),
			Heartbeat: 5 * time.Second,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ server: %v", err)
	}

	qm.UpdateConnection(conn)
	logger.Info("Connected to RabbitMQ server")
	qm.infoLog.Println("Connected to RabbitMQ.")
	return conn, nil
}


func (qm *QueueManager) HandleReInit(conn *amqp091.Connection, logger *utility.Logger) bool {
	for {
		qm.mu.Lock()
		qm.isReady = false
		qm.mu.Unlock()

		err := qm.Init(conn, logger)
		if err != nil {
			logger.Error("Failed to initialize channel. Retrying...")
			qm.errLog.Println("Failed to initialize channel. Retrying...")

			select {
			case <-qm.done:
				return true
			case <-qm.notifyConnClose:
				logger.Info("Connection closed. Reconnecting...")
				qm.infoLog.Println("Connection closed, reconnecting...")
				return false
			case <-time.After(reInitDelay):
			}
			continue
		}

		select {
		case <-qm.done:
			return true
		case <-qm.notifyConnClose:
			logger.Error("Connection closed, reconnecting...")
			qm.infoLog.Println("Connection closed, reconnecting...")
			return false
		case <-qm.notifyChanClose:
			logger.Error("Channel closed, re-running init...")
			qm.infoLog.Println("Channel closed, re-running init...")
		}
	}
}

func (qm *QueueManager) Init(conn *amqp091.Connection, logger *utility.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to open a channel: %v", err))
		qm.errLog.Printf("Failed to open a channel: %v", err)
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	err = ch.Confirm(false)
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		"",
		false, // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)

	if err != nil {
		return fmt.Errorf("failed to declare a queue: %v", err)
	}

	qm.UpdateChannel(ch)
	qm.mu.Lock()
	qm.isReady = true
	qm.mu.Unlock()
	logger.Info("Channel and queue initialization done.")
	qm.infoLog.Println("Channel and queue initialization done.")
	return nil
}

func (qm *QueueManager) UpdateConnection(conn *amqp091.Connection) {
	qm.connection = conn
	qm.notifyConnClose = make(chan *amqp091.Error)
	qm.connection.NotifyClose(qm.notifyConnClose)
}

func (qm *QueueManager) UpdateChannel(ch *amqp091.Channel) {
	qm.channel = ch
	qm.notifyChanClose = make(chan *amqp091.Error)
	qm.channel.NotifyClose(qm.notifyChanClose)
}

func (qm *QueueManager) Close() error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if !qm.isReady {
		return errShutdown
	}
	close(qm.done)
	err := qm.channel.Close()
	if err != nil {
		return err
	}

	err = qm.connection.Close()
	if err != nil {
		return err
	}

	qm.isReady = false
	return nil
}

func (qm *QueueManager) Publish(payload, routingKey string) error {
	// if qm == nil || qm.channel == nil {
	// 	fmt.Println(qm)
    //     return fmt.Errorf("RabbitMQ service is not initialized")
    // }

	qm.mu.Lock()
	if !qm.isReady {
		qm.mu.Unlock()
		return errNotConnected
	}
	qm.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return qm.channel.PublishWithContext(
		ctx,
		qm.config.Exchange, // Exchange
		routingKey,         // Routing key
		false,              // Mandatory
		false,              // Immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         []byte(payload),
			DeliveryMode: amqp091.Persistent,
		},
	)
}
