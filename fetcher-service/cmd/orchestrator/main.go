package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victoragudo/hotel-management-system/pkg/entities"

	"github.com/common-nighthawk/go-figure"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/grpcjson"
	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/infrastructure/queue"
	"github.com/victoragudo/hotel-management-system/fetcher-service/proto/orchestrator"
	"github.com/victoragudo/hotel-management-system/pkg/database"
	"github.com/victoragudo/hotel-management-system/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	config := loadConfig()

	applicationLogger := logger.SetupLogger("info")

	rabbitMQAddress := fmt.Sprintf("amqp://%s:%s@%s:%d/", config.RabbitmqUser, config.RabbitmqPassword, config.RabbitmqHost, config.RabbitmqPort)
	applicationLogger.Info(rabbitMQAddress)
	amqpConnection, err := amqp.Dial(rabbitMQAddress)
	if err != nil {
		applicationLogger.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			applicationLogger.Error("Failed to close RabbitMQ connection", "error", err)
		}
	}(amqpConnection)

	amqpChannel, err := amqpConnection.Channel()
	if err != nil {
		applicationLogger.Error("Failed to open a channel", "error", err)
		os.Exit(1)
	}

	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {
			applicationLogger.Error("Failed to close RabbitMQ channel", "error", err)
		}
	}(amqpChannel)

	rabbitMQPublisher, _ := queue.NewMQPublisher(
		amqpConnection,
		amqpChannel,
		config.QueueName,
	)

	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable", config.PostgresHost, config.PostgresPort, config.PostgresDB, config.PostgresUser, config.PostgresPassword)
	db, err := database.GormOpen(connectionString)
	if err != nil {
		applicationLogger.Error("db connect failed", "error", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(db, &entities.HotelData{}, &entities.ReviewData{}, &entities.HotelTranslation{}); err != nil {
		applicationLogger.Error("db migrations failed", "error", err)
		os.Exit(1)
	}

	server := &OrchestratorGRPCServer{
		config:            config,
		logger:            applicationLogger,
		rabbitMQPublisher: rabbitMQPublisher,
		db:                db,
	}

	if err := server.Start(); err != nil {
		applicationLogger.Error("Failed to start orchestrator server", "error", err)
		os.Exit(1)
	}
}

func (s *OrchestratorGRPCServer) Start() error {
	grpcjson.Register()
	figure.NewFigure("ORCHESTRATOR", "", true).Print()
	fmt.Println("gRPC server started at ", s.config.ServerHost, ":", s.config.ServerPost)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.ServerHost, s.config.ServerPost))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.ForceServerCodec(grpcjson.Codec{}))
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())
	reflection.Register(grpcServer)
	orchestrator.RegisterOrchestratorServiceServer(grpcServer, s)

	go func() {
		s.logger.Info(fmt.Sprintf("Starting gRPC server at %s", listener.Addr().String()))
		if err := grpcServer.Serve(listener); err != nil {
			s.logger.Error("gRPC server failed", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.runOnce(ctx)

	s.logger.Info("Orchestrator started")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	s.logger.Info("Shutting down orchestrator")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Server stopped gracefully")
	case <-shutdownCtx.Done():
		s.logger.Warn("Server stop timed out, forcing shutdown")
		grpcServer.Stop()
	}

	return nil
}
