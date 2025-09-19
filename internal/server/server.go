package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcp-tools-server/internal/config"
)

// Server combines MCP and HTTP servers
type Server struct {
	config     *config.ServerConfig
	mcpServer  *MCPServer
	httpServer *HTTPServer
}

// NewServer creates a new combined server
func NewServer(cfg *config.ServerConfig, mcpServer *MCPServer, httpServer *HTTPServer) *Server {
	return &Server{
		config:     cfg,
		mcpServer:  mcpServer,
		httpServer: httpServer,
	}
}

// Start begins both servers and handles graceful shutdown
func (s *Server) Start(ctx context.Context) error {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create error channel for MCP server
	mcpErrChan := make(chan error, 1)
	go func() {
		mcpErrChan <- s.mcpServer.Start(ctx)
	}()

	httpErrChan := make(chan error, 1)
	go func() {
		httpErrChan <- s.httpServer.Start()
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		cancel()
		return s.shutdown(ctx)
	case err := <-httpErrChan:
		cancel()
		return fmt.Errorf("HTTP server error: %w", err)
	case err := <-mcpErrChan:
		return fmt.Errorf("MCP server error: %w", err)
	}
}

// shutdown gracefully stops both servers (if running)
func (s *Server) shutdown(ctx context.Context) error {
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, time.Duration(s.config.ShutdownTimeout)*time.Second)
	defer shutdownCancel()

	// Always try to stop HTTP server if present
	if s.httpServer != nil {
		if err := s.httpServer.Stop(shutdownCtx); err != nil {
			return fmt.Errorf("failed to stop HTTP server: %w", err)
		}
	}

	return nil
}
