package main

import (
	"fmt"
	"os"

	"github.com/victoragudo/hotel-management-system/pkg/entities"

	"github.com/victoragudo/hotel-management-system/pkg/database"
	"github.com/victoragudo/hotel-management-system/pkg/logger"
)

func main() {
	config := loadConfig()
	applicationLogger := logger.SetupLogger("info")

	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable", config.PostgresHost, config.PostgresPort, config.PostgresDB, config.PostgresUser, config.PostgresPassword)
	db, err := database.GormOpen(connectionString)
	if err != nil {
		applicationLogger.Error("db connect failed", "error", err.Error())
		os.Exit(1)
	}

	if err := database.RunMigrations(db, &entities.HotelData{}, &entities.ReviewData{}, &entities.HotelTranslation{}); err != nil {
		applicationLogger.Error("db migrations failed", "error", err.Error())
		os.Exit(1)
	}

	server, err := NewMessageProcessor(config, db, applicationLogger)
	if err != nil {
		applicationLogger.Error("Failed to create message processor", "error", err.Error())
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		applicationLogger.Error("Failed to start worker server", "error", err.Error())
		os.Exit(1)
	}
}
