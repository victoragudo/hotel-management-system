package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type EntityTTLConfig struct {
	LockSeconds       int `mapstructure:"lock_seconds"`
	CacheSeconds      int `mapstructure:"cache_seconds"`
	NextUpdateSeconds int `mapstructure:"next_update_seconds"`
}

type TTLConfig struct {
	Hotels       EntityTTLConfig `mapstructure:"hotels"`
	Reviews      EntityTTLConfig `mapstructure:"reviews"`
	Translations EntityTTLConfig `mapstructure:"translations"`
}

type Config struct {
	PostgresHost     string `mapstructure:"postgres_host"`
	PostgresPort     int    `mapstructure:"postgres_port"`
	PostgresDB       string `mapstructure:"postgres_db"`
	PostgresUser     string `mapstructure:"postgres_user"`
	PostgresPassword string `mapstructure:"postgres_password"`

	RabbitmqHost     string `mapstructure:"rabbitmq_host"`
	RabbitmqPort     int    `mapstructure:"rabbitmq_port"`
	RabbitmqUser     string `mapstructure:"rabbitmq_user"`
	RabbitmqPassword string `mapstructure:"rabbitmq_password"`

	MainQueue        string `mapstructure:"main_queue"`
	MaxRetryAttempts int    `mapstructure:"max_retry_attempts"`

	RedisHost     string `mapstructure:"redis_host"`
	RedisPort     int    `mapstructure:"redis_port"`
	RedisPassword string `mapstructure:"redis_password"`

	TTL           TTLConfig `mapstructure:"ttl"`
	PrefetchCount int       `mapstructure:"prefetch_count"`

	CupidAPIURL           string `mapstructure:"cupid_api_url"`
	CupidAPIKey           string `mapstructure:"cupid_api_key"`
	CupidMaxRetryAttempts int    `mapstructure:"cupid_max_retry_attempts"`
	APITimeoutSeconds     int    `mapstructure:"api_timeout_seconds"`

	CircuitBreakerMaxFailures  int `mapstructure:"circuit_breaker_max_failures"`
	CircuitBreakerResetSeconds int `mapstructure:"circuit_breaker_reset_seconds"`
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
	if err := viper.UnmarshalKey("worker", &config); err != nil {
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

	config.CupidAPIKey = os.ExpandEnv(config.CupidAPIKey)

	fmt.Println(config.CupidAPIKey)

	config.RedisHost = os.ExpandEnv(config.RedisHost)
	config.RedisPassword = os.ExpandEnv(config.RedisPassword)
	return config
}
