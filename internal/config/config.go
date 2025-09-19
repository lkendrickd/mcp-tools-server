package config

import (
	"os"
	"strconv"
)

// ServerConfig holds the configuration for the MCP tools server
type ServerConfig struct {
	HTTPPort        int  // Port for HTTP API server
	ShutdownTimeout int  // Timeout for graceful shutdown (seconds)
	IsLocal         bool // true if running in local development mode
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

// getEnvBool reads a bool from the environment or returns the default
func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

// NewServerConfig creates a new server configuration using environment variables or defaults
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		HTTPPort:        getEnvInt("HTTP_PORT", 8080),
		ShutdownTimeout: getEnvInt("SHUTDOWN_TIMEOUT", 30),
		IsLocal:         getEnvBool("IS_LOCAL", false),
	}
}
