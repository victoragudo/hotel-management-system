package main

import (
	"strings"

	"github.com/spf13/viper"
	gotenv "github.com/subosito/gotenv"
)

type Config struct {
	IntervalsInMinutes struct {
		UpdateHotels             uint64 `mapstructure:"update_hotels"`
		UpdateReviews            uint64 `mapstructure:"update_reviews"`
		UpdateTranslations       uint64 `mapstructure:"update_translations"`
		FetchMissingTranslations uint64 `mapstructure:"fetch_missing_translations"`
		FetchMissingReviews      uint64 `mapstructure:"fetch_missing_reviews"`
	} `mapstructure:"intervals_in_minutes"`
	OrchestratorGrpcHost string `mapstructure:"orchestrator_grpc_host"`
	OrchestratorGrpcPort uint16 `mapstructure:"orchestrator_grpc_port"`
}

func loadConfig() Config {
	var err error
	if err = gotenv.Load("../.env"); err != nil {
		_ = gotenv.Load()
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("..")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	var config Config
	if !viper.IsSet("scheduler") {
		panic("scheduler section not found in config")
	}

	if err := viper.UnmarshalKey("scheduler", &config); err != nil {
		panic(err)
	}

	// Override config values with environment variables if running in Docker
	config.OrchestratorGrpcHost = viper.GetString("scheduler.orchestrator_grpc_host")
	config.OrchestratorGrpcPort = uint16(viper.GetInt("scheduler.orchestrator_grpc_port"))

	return config
}
