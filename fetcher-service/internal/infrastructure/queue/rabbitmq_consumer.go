package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sony/gobreaker"
)

type ConsumerPort interface {
	Consume() (<-chan amqp.Delivery, error)
	Close() error
	HealthCheck() error
}

type RabbitMQConfig struct {
	Host                 string
	Port                 int
	Username             string
	Password             string
	QueueName            string
	PrefetchCount        int
	MaxRetryAttempts     int
	RetryBaseDelay       time.Duration
	MaxRetryDelay        time.Duration
	ConnectionTimeout    time.Duration
	HeartbeatInterval    time.Duration
	ReconnectInterval    time.Duration
	MaxReconnectAttempts int
}

type RabbitMQConsumer struct {
	config         *RabbitMQConfig
	logger         *slog.Logger
	conn           *amqp.Connection
	channel        *amqp.Channel
	circuitBreaker *gobreaker.CircuitBreaker
	mu             sync.RWMutex
	closed         int64
	ctx            context.Context
	cancel         context.CancelFunc
	reconnectCount int64
}

func NewRabbitMQConfigFromWorkerConfig(host, username, password, queueName string, port, prefetchCount, maxRetryAttempts int) *RabbitMQConfig {
	return &RabbitMQConfig{
		Host:                 host,
		Port:                 port,
		Username:             username,
		Password:             password,
		QueueName:            queueName,
		PrefetchCount:        prefetchCount,
		MaxRetryAttempts:     maxRetryAttempts,
		RetryBaseDelay:       time.Second,
		MaxRetryDelay:        30 * time.Second,
		ConnectionTimeout:    10 * time.Second,
		HeartbeatInterval:    10 * time.Second,
		ReconnectInterval:    5 * time.Second,
		MaxReconnectAttempts: 5,
	}
}
func NewRabbitMQConsumer(config *RabbitMQConfig, logger *slog.Logger) *RabbitMQConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	cbSettings := gobreaker.Settings{
		Name:        "rabbitmq-connection",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Circuit breaker state changed",
				"name", name,
				"from", from,
				"to", to)
		},
	}

	consumer := &RabbitMQConsumer{
		config:         config,
		logger:         logger,
		circuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
		ctx:            ctx,
		cancel:         cancel,
	}

	if err := consumer.connect(); err != nil {
		logger.Error("Initial connection failed", "error", err)
	}

	go consumer.healthCheckLoop()
	go consumer.reconnectLoop()

	return consumer
}

func (c *RabbitMQConsumer) connect() error {
	return c.connectWithRetry(c.config.MaxReconnectAttempts)
}

func (c *RabbitMQConsumer) connectWithRetry(maxAttempts int) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if atomic.LoadInt64(&c.closed) == 1 {
			return fmt.Errorf("consumer is closed")
		}

		err := c.doConnect()
		if err == nil {
			c.logger.Info("Successfully connected to RabbitMQ", "attempt", attempt)
			atomic.StoreInt64(&c.reconnectCount, 0)
			return nil
		}

		lastErr = err
		atomic.AddInt64(&c.reconnectCount, 1)

		c.logger.Warn("Connection attempt failed",
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"error", err)

		if attempt < maxAttempts {
			delay := c.calculateBackoffDelay(attempt)
			select {
			case <-c.ctx.Done():
				return c.ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", maxAttempts, lastErr)
}

func (c *RabbitMQConsumer) doConnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		_ = c.conn.Close()
	}

	connStr := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		c.config.Username,
		c.config.Password,
		c.config.Host,
		c.config.Port)

	config := amqp.Config{
		Heartbeat: c.config.HeartbeatInterval,
		Dial:      amqp.DefaultDial(c.config.ConnectionTimeout),
	}

	conn, err := amqp.DialConfig(connStr, config)
	if err != nil {
		return fmt.Errorf("failed to dial RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.Qos(c.config.PrefetchCount, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	c.conn = conn
	c.channel = ch

	go c.watchConnection()

	return nil
}

func (c *RabbitMQConsumer) Consume() (<-chan amqp.Delivery, error) {
	if atomic.LoadInt64(&c.closed) == 1 {
		return nil, fmt.Errorf("consumer is closed")
	}

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		if c.channel == nil {
			return nil, fmt.Errorf("channel is not available")
		}

		deliveries, err := c.channel.Consume(
			c.config.QueueName,
			"",
			false,
			false,
			false,
			false,
			nil,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to start consuming: %w", err)
		}

		wrappedChan := make(chan amqp.Delivery)
		go func() {
			defer close(wrappedChan)
			for delivery := range deliveries {
				wrappedChan <- delivery
			}
		}()

		return wrappedChan, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(chan amqp.Delivery), nil
}

func (c *RabbitMQConsumer) Close() error {
	if !atomic.CompareAndSwapInt64(&c.closed, 0, 1) {
		return nil
	}

	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close channel: %w", err))
		}
	}

	if c.conn != nil && !c.conn.IsClosed() {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	c.logger.Info("RabbitMQ consumer closed successfully")
	return nil
}

func (c *RabbitMQConsumer) HealthCheck() error {
	if atomic.LoadInt64(&c.closed) == 1 {
		return fmt.Errorf("consumer is closed")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil || c.conn.IsClosed() {
		return fmt.Errorf("connection is not available")
	}

	if c.channel == nil {
		return fmt.Errorf("channel is not available")
	}

	return nil
}

func (c *RabbitMQConsumer) calculateBackoffDelay(attempt int) time.Duration {
	delay := c.config.RetryBaseDelay * time.Duration(1<<uint(attempt-1))
	if delay > c.config.MaxRetryDelay {
		delay = c.config.MaxRetryDelay
	}
	return delay
}

func (c *RabbitMQConsumer) watchConnection() {
	closeChan := make(chan *amqp.Error)
	c.conn.NotifyClose(closeChan)

	select {
	case err := <-closeChan:
		if err != nil {
			c.logger.Warn("Connection closed unexpectedly", "error", err)
		}
	case <-c.ctx.Done():
		return
	}
}

func (c *RabbitMQConsumer) reconnectLoop() {
	ticker := time.NewTicker(c.config.ReconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if atomic.LoadInt64(&c.closed) == 1 {
				return
			}

			c.mu.RLock()
			needReconnect := c.conn == nil || c.conn.IsClosed() || c.channel == nil
			c.mu.RUnlock()

			if needReconnect {
				c.logger.Info("Attempting to reconnect...")

				if err := c.connectWithRetry(c.config.MaxReconnectAttempts); err != nil {
					c.logger.Error("Reconnection failed", "error", err)
				}
			}

		case <-c.ctx.Done():
			return
		}
	}
}

func (c *RabbitMQConsumer) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.HealthCheck(); err != nil {
				c.logger.Warn("Health check failed", "error", err)
			} else {
				c.logger.Debug("Health check passed")
			}

		case <-c.ctx.Done():
			return
		}
	}
}
