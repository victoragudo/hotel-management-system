package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn         *amqp.Connection
	ch           *amqp.Channel
	primaryQueue string
}

type Message struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

func NewMQPublisher(amqpConnection *amqp.Connection, amqpChannel *amqp.Channel, queueName string) (*RabbitMQPublisher, error) {
	if err := amqpChannel.Confirm(false); err != nil {
		_ = amqpChannel.Close()
		_ = amqpConnection.Close()
		return nil, fmt.Errorf("failed to enable publish confirms: %w", err)
	}

	return &RabbitMQPublisher{
		conn:         amqpConnection,
		ch:           amqpChannel,
		primaryQueue: queueName,
	}, nil
}

func (p *RabbitMQPublisher) PublishBatch(ctx context.Context, messages []Message) error {
	for _, message := range messages {
		b, _ := json.Marshal(message)
		pub := amqp.Publishing{ContentType: "application/json", Body: b, DeliveryMode: amqp.Persistent, Timestamp: time.Now()}
		if err := p.ch.PublishWithContext(ctx, "", p.primaryQueue, false, false, pub); err != nil {
			return err
		}
	}
	return nil
}

func (p *RabbitMQPublisher) Close() {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

func (p *RabbitMQPublisher) backoffDelay(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 1 * time.Second
	case 2:
		return 5 * time.Second
	case 3:
		return 30 * time.Second
	case 4:
		return 2 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func (p *RabbitMQPublisher) PublishWithRetry(ctx context.Context, jobs []Message, maxAttempts int) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = p.PublishBatch(ctx, jobs)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled: %w", err)
		case <-time.After(p.backoffDelay(attempt)):
		}
	}
	return fmt.Errorf("publish failed after %d attempts: %w", maxAttempts, err)
}
