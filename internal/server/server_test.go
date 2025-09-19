package server

import (
	"context"
	"testing"
	"time"

	"log/slog"
	"mcp-tools-server/internal/config"
	"mcp-tools-server/pkg/tools"
	"os"
)

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cfg := config.NewServerConfig()
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)
	httpServer := NewHTTPServer(mcpServer, cfg.HTTPPort, logger)

	server := NewServer(cfg, mcpServer, httpServer)

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
}

func TestHTTPServer_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	// Use a different port to avoid conflicts
	httpServer := NewHTTPServer(mcpServer, 0, logger) // Port 0 lets OS choose

	// Test Start in a goroutine since it blocks
	errChan := make(chan error, 1)
	go func() {
		errChan <- httpServer.Start()
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test Stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := httpServer.Stop(ctx)
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Check that Start returned an error (because we stopped it)
	select {
	case err := <-errChan:
		// This is expected - server was shut down
		if err == nil {
			t.Error("Expected Start to return an error after shutdown")
		}
	case <-time.After(1 * time.Second):
		t.Error("Start did not return after Stop was called")
	}
}

func TestServer_shutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cfg := &config.ServerConfig{
		HTTPPort:        8080,
		ShutdownTimeout: 5,
		IsLocal:         false, // Not local, so HTTP server should be shut down
	}
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)
	httpServer := NewHTTPServer(mcpServer, cfg.HTTPPort, logger)

	server := NewServer(cfg, mcpServer, httpServer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test shutdown - this should not error even if servers aren't running
	err := server.shutdown(ctx)
	if err != nil {
		t.Errorf("shutdown failed: %v", err)
	}
}

func TestServer_shutdown_LocalMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cfg := &config.ServerConfig{
		HTTPPort:        8080,
		ShutdownTimeout: 5,
		IsLocal:         true, // Local mode, HTTP server should not be shut down
	}
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)
	httpServer := NewHTTPServer(mcpServer, cfg.HTTPPort, logger)

	server := NewServer(cfg, mcpServer, httpServer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test shutdown in local mode
	err := server.shutdown(ctx)
	if err != nil {
		t.Errorf("shutdown failed in local mode: %v", err)
	}
}
