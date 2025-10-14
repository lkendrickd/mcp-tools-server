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

// Server combines MCP, HTTP, and Streamable HTTP servers.
type Server struct {
	config               *config.ServerConfig
	mcpServer            *MCPServer
	httpServer           *HTTPServer
	streamableHTTPServer *StreamableHTTPServer
	webSocketServer      *WebSocketServer
}

// NewServer creates a new combined server.
func NewServer(
	cfg *config.ServerConfig,
	mcpServer *MCPServer,
	httpServer *HTTPServer,
	streamableHTTPServer *StreamableHTTPServer,
	webSocketServer *WebSocketServer,
) *Server {
	return &Server{
		config:               cfg,
		mcpServer:            mcpServer,
		httpServer:           httpServer,
		streamableHTTPServer: streamableHTTPServer,
		webSocketServer:      webSocketServer,
	}
}

// Start begins all configured servers and handles graceful shutdown.
func (s *Server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 4) // One for each potential server

	if s.mcpServer != nil {
		go func() {
			errChan <- s.mcpServer.Start(ctx)
		}()
	}

	if s.httpServer != nil {
		go func() {
			errChan <- s.httpServer.Start()
		}()
	}

	if s.streamableHTTPServer != nil {
		go func() {
			errChan <- s.streamableHTTPServer.Start()
		}()
	}

	if s.webSocketServer != nil {
		go func() {
			errChan <- s.webSocketServer.Start()
		}()
	}

	// Wait for a shutdown signal or a server error.
	select {
	case <-sigChan:
		cancel()
		return s.shutdown(context.Background()) // Use a new context for shutdown
	case err := <-errChan:
		cancel()
		return fmt.Errorf("server error: %w", err)
	}
}

// shutdown gracefully stops all running servers.
func (s *Server) shutdown(ctx context.Context) error {
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, time.Duration(s.config.ShutdownTimeout)*time.Second)
	defer shutdownCancel()

	var shutdownError error

	if s.httpServer != nil {
		if err := s.httpServer.Stop(shutdownCtx); err != nil {
			shutdownError = fmt.Errorf("failed to stop HTTP server: %w", err)
		}
	}

	if s.streamableHTTPServer != nil {
		if err := s.streamableHTTPServer.Stop(shutdownCtx); err != nil {
			if shutdownError != nil {
				shutdownError = fmt.Errorf("%v; failed to stop Streamable HTTP server: %w", shutdownError, err)
			} else {
				shutdownError = fmt.Errorf("failed to stop Streamable HTTP server: %w", err)
			}
		}
	}

	if s.webSocketServer != nil {
		if err := s.webSocketServer.Stop(shutdownCtx); err != nil {
			if shutdownError != nil {
				shutdownError = fmt.Errorf("%v; failed to stop WebSocket server: %w", shutdownError, err)
			} else {
				shutdownError = fmt.Errorf("failed to stop WebSocket server: %w", err)
			}
		}
	}

	// The MCP server is managed by the context passed to its Start method,
	// so it doesn't need an explicit stop call here.

	return shutdownError
}
