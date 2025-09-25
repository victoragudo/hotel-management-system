package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	ServerHost string `mapstructure:"server_host"`
	ServerPost uint16 `mapstructure:"server_port"`

	PostgresHost     string `mapstructure:"postgres_host"`
	PostgresPort     int    `mapstructure:"postgres_port"`
	PostgresDB       string `mapstructure:"postgres_db"`
	PostgresUser     string `mapstructure:"postgres_user"`
	PostgresPassword string `mapstructure:"postgres_password"`

	RabbitmqHost     string `mapstructure:"rabbitmq_host"`
	RabbitmqPort     int    `mapstructure:"rabbitmq_port"`
	RabbitmqUser     string `mapstructure:"rabbitmq_user"`
	RabbitmqPassword string `mapstructure:"rabbitmq_password"`

	QueueName        string `mapstructure:"main_queue"`
	MaxRetryAttempts int    `mapstructure:"max_retry_attempts"`

	BatchSize    int `mapstructure:"batch_size"`
	BatchDelayMs int `mapstructure:"batch_delay_ms"`
}

func loadConfig() Config {
	var err error
	if err = gotenv.Load("../.env"); err != nil {
		_ = gotenv.Load()
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("..")

	// Allow overriding nested keys via environment variables using underscores
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	var config Config
	if err := viper.UnmarshalKey("orchestrator", &config); err != nil {
		panic(err)
	}

	config.PostgresUser = os.ExpandEnv(config.PostgresUser)
	config.PostgresHost = os.ExpandEnv(config.PostgresHost)
	config.PostgresPassword = os.ExpandEnv(config.PostgresPassword)
	config.PostgresPort, _ = strconv.Atoi(os.ExpandEnv(fmt.Sprintf("%d", config.PostgresPort)))

	config.RabbitmqHost = os.ExpandEnv(config.RabbitmqHost)
	config.RabbitmqUser = os.ExpandEnv(config.RabbitmqUser)
	config.RabbitmqPassword = os.ExpandEnv(config.RabbitmqPassword)
	config.RabbitmqPort, _ = strconv.Atoi(os.ExpandEnv(fmt.Sprintf("%d", config.RabbitmqPort)))

	return config
}
