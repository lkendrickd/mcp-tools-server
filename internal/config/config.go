package config

import (
	"os"
	"strconv"
)

// ServerConfig holds the configuration for the MCP tools server
type ServerConfig struct {
	HTTPPort        int // Port for HTTP API server
	ShutdownTimeout int // Timeout for graceful shutdown (seconds)
}

// getEnvInt reads an int from the environment or returns the default
func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

// NewServerConfig creates a new server configuration using environment variables or defaults
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		HTTPPort:        getEnvInt("HTTP_PORT", 8080),
		ShutdownTimeout: getEnvInt("SHUTDOWN_TIMEOUT", 30),
	}
}
