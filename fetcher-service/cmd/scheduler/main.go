package main

import (
	"os"

	"github.com/victoragudo/hotel-management-system/pkg/logger"
)

func main() {
	config := loadConfig()

	applicationLogger := logger.SetupLogger("info")

	jobScheduler, err := NewScheduler(config, applicationLogger)
	if err != nil {
		applicationLogger.Error("Failed to create scheduler jobScheduler", "error", err)
		os.Exit(1)
	}

	jobScheduler.Start()
}
