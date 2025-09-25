package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Typesense TypesenseConfig `mapstructure:"typesense"`
	CupidAPI  CupidAPIConfig  `mapstructure:"cupid_api"`
	Sync      SyncConfig      `mapstructure:"sync"`
}

type ServerConfig struct {
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
	EnableCORS     bool          `mapstructure:"enable_cors"`
	TrustedProxies []string      `mapstructure:"trusted_proxies"`
}

type DatabaseConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	Username           string        `mapstructure:"username"`
	Password           string        `mapstructure:"password"`
	Database           string        `mapstructure:"database"`
	SSLMode            string        `mapstructure:"ssl_mode"`
	MaxOpenConnections int           `mapstructure:"max_open_connections"`
	MaxIdleConnections int           `mapstructure:"max_idle_connections"`
	ConnMaxLife        time.Duration `mapstructure:"conn_max_life"`
	Timeout            time.Duration `mapstructure:"timeout"`
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	Database     int           `mapstructure:"database"`
	PoolSize     int           `mapstructure:"pool_size"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type TypesenseConfig struct {
	ApiKey         string `mapstructure:"api_key"`
	Host           string `mapstructure:"host"`
	CollectionName string `mapstructure:"collection_name"`
}

type CupidAPIConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	APIKey  string        `mapstructure:"api_key"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type SyncConfig struct {
	BatchSize           int           `mapstructure:"batch_size"`
	InitialSyncOnStart  bool          `mapstructure:"initial_sync_on_start"`
	IncrementalInterval time.Duration `mapstructure:"incremental_interval"`
	FullSyncInterval    time.Duration `mapstructure:"full_sync_interval"`
	ConcurrentWorkers   int           `mapstructure:"concurrent_workers"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json or text
	OutputFile string `mapstructure:"output_file"`
}

func LoadConfig() (*Config, error) {
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
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.UnmarshalKey("search", &config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	expandConfigEnvVars(&config)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func expandConfigEnvVars(config *Config) {
	config.Server.Host = os.ExpandEnv(config.Server.Host)

	config.Database.Host = os.ExpandEnv(config.Database.Host)
	config.Database.Username = os.ExpandEnv(config.Database.Username)
	config.Database.Password = os.ExpandEnv(config.Database.Password)
	config.Database.Database = os.ExpandEnv(config.Database.Database)
	config.Database.SSLMode = os.ExpandEnv(config.Database.SSLMode)

	config.Redis.Host = os.ExpandEnv(config.Redis.Host)
	config.Redis.Password = os.ExpandEnv(config.Redis.Password)

	config.Typesense.ApiKey = os.ExpandEnv(config.Typesense.ApiKey)
	config.Typesense.Host = os.ExpandEnv(config.Typesense.Host)

	config.CupidAPI.BaseURL = os.ExpandEnv(config.CupidAPI.BaseURL)
	config.CupidAPI.APIKey = os.ExpandEnv(config.CupidAPI.APIKey)
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode)
}

func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *Config) Validate() error {
	if c.CupidAPI.APIKey == "" {
		return fmt.Errorf("cupid API key is required")
	}

	if c.CupidAPI.BaseURL == "" {
		return fmt.Errorf("cupid API base URL is required")
	}

	if !strings.HasPrefix(c.CupidAPI.BaseURL, "http://") && !strings.HasPrefix(c.CupidAPI.BaseURL, "https://") {
		c.CupidAPI.BaseURL = "https://" + c.CupidAPI.BaseURL
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Typesense.ApiKey == "" {
		return fmt.Errorf("typesense API key is required")
	}

	if c.Typesense.Host == "" {
		return fmt.Errorf("typesense index name is required")
	}

	if c.Typesense.CollectionName == "" {
		return fmt.Errorf("typesense index name is required")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	return nil
}
