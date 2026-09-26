package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	config "ingestion-go/config"
	"ingestion-go/pkg/models"
	"time"
)

// RabbitPublisher handles RabbitMQ publishing operations.
type RabbitPublisher struct {
	Conn         *amqp.Connection
	Ch           *amqp.Channel
	Queue        amqp.Queue
	ExchangeName string
	RoutingKey   string
}

// NewRabbitPublisher creates a new RabbitPublisher instance, establishing a connection to RabbitMQ and declaring the specified queue.
func NewRabbitPublisher(cfg config.Config) (*RabbitPublisher, error) {

	// create a new connection to RabbitMQ
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// create a new channel for publishing messages
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	// create a exchange for route messages
	exchangeName := cfg.ExchangeName
	exchangeType := cfg.ExchangeType
	err = ch.ExchangeDeclare(
		exchangeName, // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare a durable queue to prevent message loss on RabbitMQ restarts
	q, err := ch.QueueDeclare(
		cfg.QueueName, // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare RabbitMQ queue: %w", err)
	}

	// bind queue to exchange
	routingKey := cfg.RoutingKey

	err = ch.QueueBind(
		q.Name,
		routingKey,
		exchangeName,
		false, // no wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	return &RabbitPublisher{
		Conn:         conn,
		Ch:           ch,
		Queue:        q,
		ExchangeName: exchangeName,
		RoutingKey:   routingKey,
	}, nil
}

// PublishNotification serializes and sends a single notification event to RabbitMQ.
func (r *RabbitPublisher) PublishNotification(ctx context.Context, info models.BucketNotificationInfo) error {

	// Serialize the notification info to JSON
	body, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal notification to JSON: %w", err)
	}

	// Create a timeout context for the publish operation
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Publish the message to the RabbitMQ queue
	err = r.Ch.PublishWithContext(
		pubCtx,
		r.ExchangeName, // custom exchange name
		r.RoutingKey,   // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Mark message as persistent on disk
			ContentType:  "application/json",
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to RabbitMQ: %w", err)
	}

	return nil
}

// Close gracefully closes the RabbitMQ channel and connection.
func (r *RabbitPublisher) Close() {
	if r.Ch != nil {
		r.Ch.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
