package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	config "ingestion-go/config"
	"ingestion-go/pkg/storage"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitPublisher handles RabbitMQ publishing operations.
type RabbitPublisher struct {
	conn  *amqp.Connection
	ch    *amqp.Channel
	queue amqp.Queue
}

// NewRabbitPublisher creates a new RabbitPublisher instance, establishing a connection to RabbitMQ and declaring the specified queue.
func NewRabbitPublisher(cfg config.Config, amqpURL, queueName string) (*RabbitPublisher, error) {

	// create a new connection to RabbitMQ
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// create a new channel for publishing messages
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	// Declare a durable queue to prevent message loss on RabbitMQ restarts
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare RabbitMQ queue: %w", err)
	}

	return &RabbitPublisher{
		conn:  conn,
		ch:    ch,
		queue: q,
	}, nil
}

// PublishNotification serializes and sends a single notification event to RabbitMQ.
func (r *RabbitPublisher) PublishNotification(ctx context.Context, info storage.BucketNotificationInfo) error {

	// Serialize the notification info to JSON
	body, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal notification to JSON: %w", err)
	}

	// Create a timeout context for the publish operation
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Publish the message to the RabbitMQ queue
	err = r.ch.PublishWithContext(
		pubCtx,
		"",           // exchange (empty string uses default exchange)
		r.queue.Name, // routing key (matches queue name for default exchange)
		false,        // mandatory
		false,        // immediate
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
	if r.ch != nil {
		r.ch.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
