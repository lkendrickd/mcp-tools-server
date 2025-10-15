package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ServerConfig holds the configuration for the MCP tools server
type ServerConfig struct {
	HTTPPort           int      // Port for HTTP API server
	StreamableHTTPPort int      // Port for Streamable HTTP MCP server
	WebSocketPort      int      // Port for WebSocket server
	ShutdownTimeout    int      // Timeout for graceful shutdown (seconds)
	EnableOriginCheck  bool     // Whether to enforce origin check for streamable server
	AllowedOrigins     []string // Comma-separated list of allowed origins
	// KeepAlive settings (seconds) for streamable and stdio MCP servers.
	StreamableKeepAliveSeconds int
	StdioKeepAliveSeconds      int
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

// getEnvStringSlice reads a comma-separated string from the environment or returns the default
func getEnvStringSlice(key string, defaultVal []string) []string {
	if val, ok := os.LookupEnv(key); ok {
		if val != "" {
			return strings.Split(val, ",")
		}
	}
	return defaultVal
}

// NewServerConfig creates a new server configuration using environment variables or defaults
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		HTTPPort:                   getEnvInt("HTTP_PORT", 8080),
		StreamableHTTPPort:         getEnvInt("STREAMABLE_HTTP_PORT", 8081),
		WebSocketPort:              getEnvInt("WEBSOCKET_PORT", 8082),
		ShutdownTimeout:            getEnvInt("SHUTDOWN_TIMEOUT", 30),
		EnableOriginCheck:          getEnvBool("ENABLE_ORIGIN_CHECK", false),
		AllowedOrigins:             getEnvStringSlice("ALLOWED_ORIGINS", []string{"*"}),
		StreamableKeepAliveSeconds: getEnvInt("STREAMABLE_KEEPALIVE_SECONDS", 60),
		StdioKeepAliveSeconds:      getEnvInt("STDIO_KEEPALIVE_SECONDS", 60),
	}
}

// WebSocketAddr returns the address for the WebSocket server
func (c *ServerConfig) WebSocketAddr() string {
	return fmt.Sprintf(":%d", c.WebSocketPort)
}
