package config

import (
	"log"
	"time"

	"github.com/BurntSushi/toml"
)

// MQTTConfig holds MQTT broker configuration
type MQTTConfig struct {
	Broker   string `toml:"broker"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	ClientID string `toml:"client_id"`
	QoS      byte   `toml:"qos"`
}

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig    `toml:"server"`
	Database  DatabaseConfig  `toml:"database"`
	JWT       JWTConfig       `toml:"jwt"`
	App       AppConfig       `toml:"app"`
	RateLimit RateLimitConfig `toml:"rate_limit"`
	MQTT      MQTTConfig      `toml:"mqtt"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string        `toml:"host"`
	Port         int           `toml:"port"`
	ReadTimeout  time.Duration `toml:"read_timeout"`
	WriteTimeout time.Duration `toml:"write_timeout"`
	IdleTimeout  time.Duration `toml:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `toml:"host"`
	Port            int           `toml:"port"`
	User            string        `toml:"user"`
	Password        string        `toml:"password"`
	DBName          string        `toml:"dbname"`
	SSLMode         string        `toml:"sslmode"`
	MaxOpenConns    int           `toml:"max_open_conns"`
	MaxIdleConns    int           `toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `toml:"conn_max_lifetime"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret             string `toml:"secret"`
	ExpireHours        int    `toml:"expire_hours"`
	RefreshExpireHours int    `toml:"refresh_expire_hours"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Environment string `toml:"environment"`
	LogLevel    string `toml:"log_level"`
	BCryptCost  int    `toml:"bcrypt_cost"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int `toml:"requests_per_minute"`
	Burst             int `toml:"burst"`
}

// Load loads configuration from TOML file
func Load(path string) (*Config, error) {
	var config Config

	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// MustLoad loads configuration or panics if error
func MustLoad(path string) *Config {
	config, err := Load(path)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return config
}
