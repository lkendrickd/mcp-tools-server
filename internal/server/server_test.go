package server

import (
	"context"
	"log/slog"
	"mcp-tools-server/internal/config"
	"mcp-tools-server/pkg/tools"
	"os"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cfg := config.NewServerConfig()
	registry := tools.NewToolRegistry()
	toolService, _ := NewToolService(registry, logger)
	mcpServer := NewMCPServer(toolService, logger)
	httpServer := NewHTTPServer(toolService, cfg.HTTPPort, logger)

	server := NewServer(cfg, mcpServer, httpServer, nil, nil)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.config != cfg {
		t.Error("Server does not have correct config reference")
	}
	if server.mcpServer != mcpServer {
		t.Error("Server does not have correct MCP server reference")
	}
	if server.httpServer != httpServer {
		t.Error("Server does not have correct HTTP server reference")
	}
	if server.streamableHTTPServer != nil {
		t.Error("Expected nil streamableHTTPServer")
	}
}

func TestServer_shutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cfg := &config.ServerConfig{
		HTTPPort:        8080,
		ShutdownTimeout: 5,
	}
	registry := tools.NewToolRegistry()
	toolService, _ := NewToolService(registry, logger)
	mcpServer := NewMCPServer(toolService, logger)
	httpServer := NewHTTPServer(toolService, cfg.HTTPPort, logger)

	server := NewServer(cfg, mcpServer, httpServer, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test shutdown - this should not error even if servers aren't running
	err := server.shutdown(ctx)
	if err != nil {
		t.Errorf("shutdown failed: %v", err)
	}
}
