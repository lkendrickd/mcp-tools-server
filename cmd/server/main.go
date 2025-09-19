package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"mcp-tools-server/internal/config"
	"mcp-tools-server/internal/server"
	"mcp-tools-server/internal/version"
	"mcp-tools-server/pkg/tools"
)

func main() {
	// Check for version flag
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("MCP Tools Server v%s\n", version.GetVersion())
		fmt.Printf("Build Time: %s\n", version.GetBuildTime())
		fmt.Printf("Git Commit: %s\n", version.GetGitCommit())
		os.Exit(0)
	}

	// Setup the slog logger
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo}),
	)
	slog.SetDefault(logger)

	// Check command line args to determine mode
	if len(os.Args) > 1 && os.Args[1] == "http-only" {
		runHTTPOnly(logger)
		return
	}

	// Default: run combined server (for MCP clients)
	runCombined(logger)
}

func runCombined(logger *slog.Logger) {
	// Initialize dependencies
	cfg := config.NewServerConfig()
	registry := tools.NewToolRegistry()

	// Create servers with the registry
	mcpServer := server.NewMCPServer(registry, logger)

	httpServer := server.NewHTTPServer(mcpServer, cfg.HTTPPort, logger)

	// Create combined server
	srv := server.NewServer(cfg, mcpServer, httpServer)

	// Start the server
	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runHTTPOnly(logger *slog.Logger) {
	// Run only HTTP server for testing
	cfg := config.NewServerConfig()
	registry := tools.NewToolRegistry()

	// Create MCP server to get access to tools
	mcpServer := server.NewMCPServer(registry, logger)

	httpServer := server.NewHTTPServer(mcpServer, cfg.HTTPPort, logger)

	logger.Info("Starting HTTP server", "port", cfg.HTTPPort)
	if err := httpServer.Start(); err != nil {
		logger.Error("HTTP server error", "error", err)
	}
}
