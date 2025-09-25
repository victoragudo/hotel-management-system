package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/google/uuid"
	"github.com/jasonlvhit/gocron"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/grpcjson"
	"github.com/victoragudo/hotel-management-system/fetcher-service/proto/orchestrator"
	"github.com/victoragudo/hotel-management-system/fetcher-service/proto/scheduler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Scheduler struct {
	config             Config
	orchestratorServer orchestrator.OrchestratorServiceClient
	scheduler          *gocron.Scheduler
	logger             *slog.Logger
}

func NewScheduler(config Config, logger *slog.Logger) (*Scheduler, error) {
	grpcjson.Register()

	addr := fmt.Sprintf("%s:%d", config.OrchestratorGrpcHost, config.OrchestratorGrpcPort)
	grpcConnection, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcjson.Codec{})))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to orchestrator: %w", err)
	}

	s := &Scheduler{
		config:             config,
		orchestratorServer: orchestrator.NewOrchestratorServiceClient(grpcConnection),
		scheduler:          gocron.NewScheduler(),
		logger:             logger,
	}

	if err := s.setupSchedules(); err != nil {
		return nil, fmt.Errorf("failed to setup schedules: %w", err)
	}

	return s, nil
}

func (s *Scheduler) Start() {
	s.scheduler.Start()
	figure.NewFigure("SCHEDULER", "", true).Print()
	s.logger.Info(fmt.Sprintf("Scheduler started, dialing at --> %s:%d", s.config.OrchestratorGrpcHost, s.config.OrchestratorGrpcPort))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	s.logger.Info("Shutting down scheduler")
	s.scheduler.Clear()
}

func (s *Scheduler) TriggerFetch(ctx context.Context, triggerRequest *scheduler.TriggerRequest) (*scheduler.TriggerResponse, error) {
	var messageType orchestrator.MessageType
	switch triggerRequest.MessageType {
	case scheduler.MessageType_UPDATE_HOTEL:
		messageType = orchestrator.MessageType_UPDATE_HOTEL
	case scheduler.MessageType_UPDATE_REVIEW:
		messageType = orchestrator.MessageType_UPDATE_REVIEW
	case scheduler.MessageType_UPDATE_TRANSLATION:
		messageType = orchestrator.MessageType_UPDATE_TRANSLATION
	case scheduler.MessageType_FETCH_MISSING_TRANSLATIONS:
		messageType = orchestrator.MessageType_FETCH_MISSING_TRANSLATIONS
	case scheduler.MessageType_FETCH_MISSING_REVIEWS:
		messageType = orchestrator.MessageType_FETCH_MISSING_REVIEWS
	default:
		messageType = orchestrator.MessageType_UNSPECIFIED
	}

	return s.triggerFetch(ctx, triggerRequest.MessageType.String(), messageType, triggerRequest)
}

func (s *Scheduler) triggerFetch(ctx context.Context, scheduleType string, messageType orchestrator.MessageType, triggerRequest *scheduler.TriggerRequest) (*scheduler.TriggerResponse, error) {
	requestID := triggerRequest.RequestId
	if requestID == "" {
		requestID = uuid.New().String()
	}

	fetchRequest := &orchestrator.FetchRequest{
		RequestId:   requestID,
		MessageType: messageType,
		Timestamp:   triggerRequest.Timestamp,
		Force:       triggerRequest.Force,
	}

	fetchResponse, err := s.orchestratorServer.ProcessFetchRequest(ctx, fetchRequest)

	if err != nil {
		s.logger.Error("Failed to trigger fetch",
			"type", scheduleType,
			"request_id", requestID,
			"error", err)
		return &scheduler.TriggerResponse{
			Success:    false,
			Message:    fmt.Sprintf("Failed to trigger %s fetch: %v", scheduleType, err),
			RequestId:  requestID,
			JobsQueued: 0,
		}, nil
	}

	if fetchResponse.JobsCreated > 0 {
		s.logger.Info("Fetch triggered successfully",
			"type", scheduleType,
			"request_id", requestID,
			"jobs_created", fetchResponse.JobsCreated)
	}
	return &scheduler.TriggerResponse{
		Success:    fetchResponse.Success,
		Message:    fetchResponse.Message,
		RequestId:  fetchResponse.RequestId,
		JobsQueued: fetchResponse.JobsCreated,
	}, nil
}

func (s *Scheduler) trigger(messageType scheduler.MessageType) {
	ctx := context.Background()
	triggerRequest := &scheduler.TriggerRequest{
		RequestId:   uuid.New().String(),
		Timestamp:   time.Now().Unix(),
		Force:       false,
		MessageType: messageType,
	}

	_, err := s.TriggerFetch(ctx, triggerRequest)
	if err != nil {
		s.logger.Error("Scheduled failed", "error", err)
	}
}

func (s *Scheduler) setupSchedules() error {
	err := s.scheduler.Every(s.config.IntervalsInMinutes.UpdateHotels).Minutes().Do(func() {
		s.trigger(scheduler.MessageType_UPDATE_HOTEL)
		s.logger.Info(
			"Triggered update hotels",
			"timestamp", time.Now().Unix(),
			"interval", s.config.IntervalsInMinutes.UpdateHotels,
		)
	})
	if err != nil {
		s.logger.Error("Failed to setup hotel fetch schedule", "error", err)
	}

	err = s.scheduler.Every(s.config.IntervalsInMinutes.UpdateReviews).Minutes().Do(func() {
		s.trigger(scheduler.MessageType_UPDATE_REVIEW)
		s.logger.Info(
			"Triggered update reviews",
			"timestamp", time.Now().Unix(),
			"interval", s.config.IntervalsInMinutes.UpdateReviews,
		)
	})
	if err != nil {
		s.logger.Error("Failed to setup review fetch schedule", "error", err)
	}

	err = s.scheduler.Every(s.config.IntervalsInMinutes.UpdateTranslations).Minutes().Do(func() {
		s.trigger(scheduler.MessageType_UPDATE_TRANSLATION)
		s.logger.Info(
			"Triggered update translations",
			"timestamp", time.Now().Unix(),
			"interval", s.config.IntervalsInMinutes.UpdateTranslations,
		)
	})
	if err != nil {
		s.logger.Error("Failed to setup translation fetch schedule", "error", err)
	}

	err = s.scheduler.Every(s.config.IntervalsInMinutes.FetchMissingTranslations).Minutes().Do(func() {
		s.trigger(scheduler.MessageType_FETCH_MISSING_TRANSLATIONS)
		s.logger.Info(
			"Triggered missing translations",
			"timestamp", time.Now().Unix(),
			"interval", s.config.IntervalsInMinutes.FetchMissingTranslations,
		)
	})
	if err != nil {
		s.logger.Error("Failed to setup missing translations schedule", "error", err)
	}

	err = s.scheduler.Every(s.config.IntervalsInMinutes.FetchMissingReviews).Minutes().Do(func() {
		s.trigger(scheduler.MessageType_FETCH_MISSING_REVIEWS)
		s.logger.Info(
			"Triggered missing reviews",
			"timestamp", time.Now().Unix(),
			"interval", s.config.IntervalsInMinutes.FetchMissingReviews,
		)
	})
	if err != nil {
		s.logger.Error("Failed to setup missing reviews schedule", "error", err)
	}

	s.logger.Info("Schedules configured",
		"update_hotels_interval", s.config.IntervalsInMinutes.UpdateHotels,
		"update_translations_interval", s.config.IntervalsInMinutes.UpdateTranslations,
		"update_reviews_interval", s.config.IntervalsInMinutes.UpdateReviews,
		"missing_reviews_schedule", s.config.IntervalsInMinutes.FetchMissingReviews,
		"missing_translations_schedule", s.config.IntervalsInMinutes.FetchMissingTranslations)

	return nil
}
