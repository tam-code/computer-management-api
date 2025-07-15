package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration with validation
type Config struct {
	// Application settings
	Port     int    `validate:"required,min=1,max=65535"`
	LogLevel string `validate:"required,oneof=debug info warn error"`

	// Database settings
	Database DatabaseConfig `validate:"required"`

	// External services
	NotificationService NotificationConfig `validate:"required"`

	// Security settings
	Security SecurityConfig `validate:"required"`

	// Performance settings
	Server ServerConfig `validate:"required"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string `validate:"required"`
	Port            int    `validate:"required,min=1,max=65535"`
	User            string `validate:"required"`
	Password        string `validate:"required"`
	Name            string `validate:"required"`
	SSLMode         string `validate:"required,oneof=disable require verify-ca verify-full"`
	MaxOpenConns    int    `validate:"min=1"`
	MaxIdleConns    int    `validate:"min=1"`
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NotificationConfig holds notification service configuration
type NotificationConfig struct {
	URL            string        `validate:"required,url"`
	Timeout        time.Duration `validate:"required"`
	RetryAttempts  int           `validate:"min=0,max=10"`
	RetryDelay     time.Duration
	MaxPayloadSize int64 `validate:"min=1024"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimitRPS    int           `validate:"min=1"`
	RateLimitBurst  int           `validate:"min=1"`
	RequestTimeout  time.Duration `validate:"required"`
	ShutdownTimeout time.Duration `validate:"required"`
	EnableCORS      bool
	AllowedOrigins  []string
	TrustedProxies  []string
}

// ServerConfig holds server performance configuration
type ServerConfig struct {
	ReadTimeout     time.Duration `validate:"required"`
	WriteTimeout    time.Duration `validate:"required"`
	IdleTimeout     time.Duration `validate:"required"`
	MaxHeaderBytes  int           `validate:"min=1024"`
	EnableMetrics   bool
	MetricsPort     int `validate:"min=1,max=65535"`
	EnableProfiling bool
}

// LoadConfig loads and validates the configuration from environment variables
func LoadConfig() (*Config, error) {

	config := &Config{
		Port:     getEnvAsInt("PORT", 8080),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", ""),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},

		NotificationService: NotificationConfig{
			URL:            getEnv("NOTIFIER_URL", ""),
			Timeout:        getEnvAsDuration("NOTIFIER_TIMEOUT", 10*time.Second),
			RetryAttempts:  getEnvAsInt("NOTIFIER_RETRY_ATTEMPTS", 3),
			RetryDelay:     getEnvAsDuration("NOTIFIER_RETRY_DELAY", time.Second),
			MaxPayloadSize: getEnvAsInt64("NOTIFIER_MAX_PAYLOAD_SIZE", 1024*1024),
		},

		Security: SecurityConfig{
			RateLimitRPS:    getEnvAsInt("RATE_LIMIT_RPS", 100),
			RateLimitBurst:  getEnvAsInt("RATE_LIMIT_BURST", 200),
			RequestTimeout:  getEnvAsDuration("REQUEST_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
			EnableCORS:      getEnvAsBool("ENABLE_CORS", true),
			AllowedOrigins:  getEnvAsSlice("ALLOWED_ORIGINS", []string{"*"}),
			TrustedProxies:  getEnvAsSlice("TRUSTED_PROXIES", []string{}),
		},

		Server: ServerConfig{
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
			MaxHeaderBytes:  getEnvAsInt("SERVER_MAX_HEADER_BYTES", 1<<20), // 1MB
			EnableMetrics:   getEnvAsBool("ENABLE_METRICS", true),
			MetricsPort:     getEnvAsInt("METRICS_PORT", 9090),
			EnableProfiling: getEnvAsBool("ENABLE_PROFILING", false),
		},
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validateConfig performs basic validation on the configuration
func validateConfig(config *Config) error {
	var errors []string

	// Validate required database fields
	if config.Database.User == "" {
		errors = append(errors, "database user is required")
	}
	if config.Database.Password == "" {
		errors = append(errors, "database password is required in production")
	}
	if config.Database.Name == "" {
		errors = append(errors, "database name is required")
	}

	// Validate notification URL
	if config.NotificationService.URL == "" {
		errors = append(errors, "notification service URL is required")
	}

	// Validate port ranges
	if config.Port < 1 || config.Port > 65535 {
		errors = append(errors, "port must be between 1 and 65535")
	}
	if config.Database.Port < 1 || config.Database.Port > 65535 {
		errors = append(errors, "database port must be between 1 and 65535")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User,
		c.Database.Password, c.Database.Name, c.Database.SSLMode)
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
