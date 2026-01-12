package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port              string
	AllowedOrigins    []string
	WSReadTimeout     time.Duration
	WSWriteTimeout    time.Duration
	LogLevel          string
	PingPeriod        time.Duration
	PongWait          time.Duration
	WriteWait         time.Duration
	MaxMessageSize    int64
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		Port:           getEnv("PORT", "8080"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173"), ","),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}

	// Parse WebSocket timeouts
	wsReadTimeout, err := strconv.Atoi(getEnv("WS_READ_TIMEOUT", "60"))
	if err != nil {
		return nil, fmt.Errorf("invalid WS_READ_TIMEOUT: %w", err)
	}
	config.WSReadTimeout = time.Duration(wsReadTimeout) * time.Second

	wsWriteTimeout, err := strconv.Atoi(getEnv("WS_WRITE_TIMEOUT", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid WS_WRITE_TIMEOUT: %w", err)
	}
	config.WSWriteTimeout = time.Duration(wsWriteTimeout) * time.Second

	// Calculate WebSocket constants
	config.PongWait = config.WSReadTimeout
	config.PingPeriod = (config.PongWait * 9) / 10 // Must be less than pongWait
	config.WriteWait = config.WSWriteTimeout
	config.MaxMessageSize = 512

	// Trim spaces from allowed origins
	for i, origin := range config.AllowedOrigins {
		config.AllowedOrigins[i] = strings.TrimSpace(origin)
	}

	return config, nil
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
