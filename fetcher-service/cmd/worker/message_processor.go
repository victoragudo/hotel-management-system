package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/common-nighthawk/go-figure"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/infrastructure/queue"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/adapter"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/dto"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/worker/ports"
	"github.com/victoragudo/hotel-management-system/fetcher-service/pkg/constants"
	constants2 "github.com/victoragudo/hotel-management-system/pkg/constants"
	"gorm.io/gorm"
)

type MessageProcessor struct {
	config           Config
	logger           *slog.Logger
	cupidAPI         ports.APIClientPort
	gormRepo         ports.RepositoryPort
	redisCache       ports.CachePort
	redisLock        ports.LockPort
	shutdownChan     chan os.Signal
	ctx              context.Context
	cancel           context.CancelFunc
	db               *gorm.DB
	rabbitMQConsumer *queue.RabbitMQConsumer
}

type queueMessage struct {
	ID          string         `json:"id"`
	MessageType string         `json:"type"`
	Data        map[string]any `json:"data"`
}

func (messageProcessor *MessageProcessor) getTTLConfigForEntity(messageType string) EntityTTLConfig {
	switch messageType {
	case constants.MessageTypeUpdateHotel:
		return messageProcessor.config.TTL.Hotels
	case constants.MessageTypeUpdateReview:
		return messageProcessor.config.TTL.Reviews
	case constants.MessageTypeUpdateTranslation:
		return messageProcessor.config.TTL.Translations
	default:
		// Default to hotels config if unknown type
		return messageProcessor.config.TTL.Hotels
	}
}

func NewMessageProcessor(config Config, db *gorm.DB, applicationLogger *slog.Logger) (*MessageProcessor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	server := &MessageProcessor{
		config:       config,
		db:           db,
		logger:       applicationLogger,
		shutdownChan: make(chan os.Signal, 1),
		ctx:          ctx,
		cancel:       cancel,
	}

	if err := server.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return server, nil
}

func (messageProcessor *MessageProcessor) initializeServices() error {
	apiConfig := &adapter.APIConfig{
		BaseURL:       messageProcessor.config.CupidAPIURL,
		APIKey:        messageProcessor.config.CupidAPIKey,
		Timeout:       time.Duration(messageProcessor.config.APITimeoutSeconds) * time.Second,
		RateLimit:     10.0,
		BurstLimit:    20,
		MaxRetries:    messageProcessor.config.CupidMaxRetryAttempts,
		RetryInterval: 1 * time.Second,
		Headers:       make(map[string]string),
		CircuitBreaker: &adapter.CircuitBreakerConfig{
			MaxRequests: uint32(messageProcessor.config.CircuitBreakerMaxFailures),
			Interval:    60 * time.Second,
			Timeout:     time.Duration(messageProcessor.config.CircuitBreakerResetSeconds) * time.Second,
		},
	}
	messageProcessor.cupidAPI = adapter.NewCupidAPIAdapter(apiConfig)

	var err error
	messageProcessor.gormRepo, err = adapter.NewGormRepository(messageProcessor.db)
	if err != nil {
		return fmt.Errorf("failed to create GORM repository: %w", err)
	}

	redisAddr := fmt.Sprintf("%s:%d", messageProcessor.config.RedisHost, messageProcessor.config.RedisPort)
	messageProcessor.redisCache = adapter.NewRedisCacheAdapter(redisAddr, messageProcessor.config.RedisPassword, 0)
	messageProcessor.redisLock = adapter.NewRedisLockAdapter(redisAddr, messageProcessor.config.RedisPassword, 0)

	rabbitMQConfig := queue.NewRabbitMQConfigFromWorkerConfig(
		messageProcessor.config.RabbitmqHost,
		messageProcessor.config.RabbitmqUser,
		messageProcessor.config.RabbitmqPassword,
		messageProcessor.config.MainQueue,
		messageProcessor.config.RabbitmqPort, messageProcessor.config.PrefetchCount, messageProcessor.config.MaxRetryAttempts,
	)
	messageProcessor.rabbitMQConsumer = queue.NewRabbitMQConsumer(rabbitMQConfig, messageProcessor.logger)

	return nil
}

func (messageProcessor *MessageProcessor) Start() error {
	signal.Notify(messageProcessor.shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		figure.NewFigure("WORKER", "", true).Print()
		messageProcessor.logger.Info("Starting message consumption")
		if err := messageProcessor.consumeMessages(); err != nil {
			messageProcessor.logger.Error("Message consumption failed", "error", err)
		}
	}()

	<-messageProcessor.shutdownChan
	messageProcessor.logger.Info("Received shutdown signal, starting graceful shutdown")

	return messageProcessor.shutdown()
}

func (messageProcessor *MessageProcessor) consumeMessages() error {
	messages, err := messageProcessor.rabbitMQConsumer.Consume()
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	for {
		select {
		case <-messageProcessor.ctx.Done():
			return nil
		case msg, ok := <-messages:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			if err := messageProcessor.processMessage(msg); err != nil {
				messageProcessor.logger.Error("Failed to process message", "error", err)
				messageProcessor.logger.Warn("Message discarded and sent to Dead Letter Queue (DLQ)",
					"message_id", string(msg.Body),
					"routing_key", msg.RoutingKey,
					"error", err)
				_ = msg.Nack(false, false)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

func (messageProcessor *MessageProcessor) processMessage(msg amqp.Delivery) error {
	var message queueMessage
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	messageProcessor.logger.Info("Processing job",
		"id", message.ID,
		"fetch_type", message.MessageType)

	lockKey := fmt.Sprintf("hotel_lock_%s", message.ID)
	entityTTL := messageProcessor.getTTLConfigForEntity(message.MessageType)
	lockTTL := time.Duration(entityTTL.LockSeconds) * time.Second
	locked, err := messageProcessor.redisLock.Acquire(messageProcessor.ctx, lockKey, lockTTL)
	if err != nil {
		messageProcessor.logger.Info(fmt.Sprintf("Redis dsn connection: %s %d %s", messageProcessor.config.RedisHost, messageProcessor.config.RedisPort, messageProcessor.config.RedisPassword))
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		messageProcessor.logger.Warn(fmt.Sprintf("%s is already being processed, skipping id %s", message.MessageType, message.ID))
		return nil
	}

	defer func() {
		if err := messageProcessor.redisLock.Release(messageProcessor.ctx, lockKey); err != nil {
			messageProcessor.logger.Error("Failed to release lock", "error", err)
		}
	}()

	var processErr error
	switch message.MessageType {
	case constants.MessageTypeUpdateHotel:
		processErr = messageProcessor.processHotelMessage(message)
	case constants.MessageTypeUpdateReview, constants.MessageTypeFetchReview:
		processErr = messageProcessor.processReviewsMessage(message)
	case constants.MessageTypeUpdateTranslation, constants.MessageTypeFetchTranslation:
		processErr = messageProcessor.processTranslationsMessage(message)
	default:
		messageProcessor.logger.Warn("Unknown fetch_type, skipping", "fetch_type", message.MessageType)
		return nil
	}
	if processErr != nil {
		return fmt.Errorf("failed to process %s job: %w", message.MessageType, processErr)
	}

	messageProcessor.logger.Info("Successfully processed job",
		"id", message.ID,
		"fetch_type", message.MessageType)

	return nil
}

func (messageProcessor *MessageProcessor) processHotelMessage(message queueMessage) error {
	cacheKey := fmt.Sprintf("hotel_data_%s", message.ID)

	var cachedData any
	found, err := messageProcessor.redisCache.Get(messageProcessor.ctx, cacheKey, &cachedData)
	if err == nil && found {
		messageProcessor.logger.Info("Using cached hotel data", "id", message.ID)
		return nil
	}

	hotelId := messageProcessor.gormRepo.GetHotelIdByPk(messageProcessor.ctx, message.ID)
	hotelAPIResponse, err := messageProcessor.cupidAPI.FetchHotelData(messageProcessor.ctx, hotelId)
	if err != nil {
		return fmt.Errorf("failed to fetch hotel data: %w", err)
	}

	hotelData, err := hotelAPIResponse.ToHotelData()
	if err != nil {
		return fmt.Errorf("failed to convert hotel data: %w", err)
	}

	hotelTTL := messageProcessor.getTTLConfigForEntity(message.MessageType)
	hotelData.NextUpdateAt = time.Now().Add(time.Duration(hotelTTL.NextUpdateSeconds) * time.Second)

	if err := messageProcessor.gormRepo.UpsertHotel(messageProcessor.ctx, hotelData); err != nil {
		return fmt.Errorf("failed to persist hotel data: %w", err)
	}

	if err := messageProcessor.redisCache.Set(messageProcessor.ctx, cacheKey, hotelAPIResponse, time.Duration(hotelTTL.CacheSeconds)*time.Second); err != nil {
		messageProcessor.logger.Warn("Failed to cache hotel data", "error", err)
	}

	messageProcessor.logger.Info(fmt.Sprintf("Successfully processed and persisted hotel data: id --> %s, next_update_at --> %s", message.ID, hotelData.NextUpdateAt.Format(time.RFC3339)))
	return nil
}

func (messageProcessor *MessageProcessor) processReviewsMessage(message queueMessage) error {
	cacheKey := fmt.Sprintf("reviews_data_%s", message.ID)
	var cached any
	found, err := messageProcessor.redisCache.Get(messageProcessor.ctx, cacheKey, &cached)
	if err == nil && found {
		messageProcessor.logger.Info("Using cached reviews", "id", message.ID)
		return nil
	}

	var hotelId int64
	var reviewCount int64

	if message.MessageType == constants.MessageTypeFetchReview {
		if message.Data == nil {
			return fmt.Errorf("message data is nil")
		}
		hotelIdStr := message.Data[constants2.HotelId].(string)
		hotelIdParsed, err := strconv.ParseInt(hotelIdStr, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse hotel_id: %w", err)
		}
		hotelId = hotelIdParsed
		reviewCount = 10
	} else {
		hotelId = messageProcessor.gormRepo.GetHotelIdFromReviewByPk(messageProcessor.ctx, message.ID)
		if hotelId == 0 {
			return nil
		}

		reviewCount = messageProcessor.gormRepo.ReviewCountByHotelId(messageProcessor.ctx, hotelId)
		if reviewCount == 0 {
			return nil
		}
	}

	fetchedReviews, err := messageProcessor.cupidAPI.FetchHotelReviews(messageProcessor.ctx, hotelId, &dto.ReviewFetchOptions{
		ReviewCount: reviewCount,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch reviews: %w", err)
	}

	mappedReviews, err := fetchedReviews.ToReviewDataList(hotelId)
	if err != nil {
		return fmt.Errorf("failed to convert reviews: %w", err)
	}

	reviewsTTL := messageProcessor.getTTLConfigForEntity("reviews")
	for _, review := range mappedReviews {
		review.NextUpdateAt = time.Now().Add(time.Duration(reviewsTTL.NextUpdateSeconds) * time.Second)
		if existing, err := messageProcessor.gormRepo.GetReviewByReviewID(messageProcessor.ctx, review.ReviewID); err == nil && existing != nil && existing.ID != "" {
			review.ID = existing.ID
			if err := messageProcessor.gormRepo.UpdateReview(messageProcessor.ctx, review); err != nil {
				return fmt.Errorf("failed to update review %d: %w", review.ReviewID, err)
			}
		} else {
			if err := messageProcessor.gormRepo.CreateReview(messageProcessor.ctx, review); err != nil {
				return fmt.Errorf("failed to create review %d: %w", review.ReviewID, err)
			}
		}
	}

	if err := messageProcessor.redisCache.Set(messageProcessor.ctx, cacheKey, fetchedReviews, time.Duration(reviewsTTL.CacheSeconds)*time.Second); err != nil {
		messageProcessor.logger.Warn("Failed to cache reviews", "error", err)
	}

	messageProcessor.logger.Info("Processed reviews", "id", message.ID, "count", len(mappedReviews))

	return nil
}

func (messageProcessor *MessageProcessor) processTranslationsMessage(message queueMessage) error {
	cacheKey := fmt.Sprintf("translations_data_%s", message.ID)

	var cachedData any
	found, err := messageProcessor.redisCache.Get(messageProcessor.ctx, cacheKey, &cachedData)
	if err == nil && found {
		messageProcessor.logger.Info("Using cached translations data", "id", message.ID)
		return nil
	}

	if message.Data == nil {
		return fmt.Errorf("message data is nil")
	}

	hotelId := message.Data[constants2.HotelId].(string)
	lang := ""
	if message.MessageType == constants.MessageTypeFetchTranslation {
		lang = message.Data[constants2.Lang].(string)
	} else if message.MessageType == constants.MessageTypeUpdateTranslation {
		lang = messageProcessor.gormRepo.GetLangById(messageProcessor.ctx, message.ID)
	}

	if lang == "" {
		return fmt.Errorf("lang is empty")
	}

	translationsAPIResponse, err := messageProcessor.cupidAPI.FetchTranslations(messageProcessor.ctx, hotelId, &dto.TranslationFetchOptions{
		Lang: lang,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch translations data: %w", err)
	}

	translationsData, err := translationsAPIResponse.ToHotelTranslations(lang)
	if err != nil {
		return fmt.Errorf("failed to convert translations data: %w", err)
	}

	translationsTTL := messageProcessor.getTTLConfigForEntity("translations")
	translationsData.NextUpdateAt = time.Now().Add(time.Duration(translationsTTL.NextUpdateSeconds) * time.Second)

	if err := messageProcessor.gormRepo.UpsertHotelTranslations(messageProcessor.ctx, translationsData); err != nil {
		return fmt.Errorf("failed to persist translations data: %w", err)
	}

	if err := messageProcessor.redisCache.Set(messageProcessor.ctx, cacheKey, translationsAPIResponse, time.Duration(translationsTTL.CacheSeconds)*time.Second); err != nil {
		messageProcessor.logger.Warn("Failed to cache translations data", "error", err)
	}
	messageProcessor.logger.Info(fmt.Sprintf("Successfully processed and persisted translations data: id --> %s, lang --> %s next_update_at --> %s", message.ID, lang, translationsData.NextUpdateAt.Format(time.RFC3339)))

	return nil
}

func (messageProcessor *MessageProcessor) shutdown() error {
	messageProcessor.logger.Info("Shutting down worker server")
	messageProcessor.cancel()

	if messageProcessor.rabbitMQConsumer != nil {
		_ = messageProcessor.rabbitMQConsumer.Close()
	}

	if messageProcessor.redisCache != nil {
		_ = messageProcessor.redisCache.Close()
	}

	if messageProcessor.redisLock != nil {
		_ = messageProcessor.redisLock.Close()
	}

	messageProcessor.logger.Info("Worker server shutdown complete")
	return nil
}
