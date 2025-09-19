package main

import (
	"context"
	"flag"
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
	// Flags for server modes
	var (
		showVersion = flag.Bool("version", false, "Show version and exit")
		enableHTTP  = flag.Bool("http", false, "Enable HTTP server")
		enableMCP   = flag.Bool("mcp", false, "Enable MCP server")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("MCP Tools Server v%s\n", version.GetVersion())
		fmt.Printf("Build Time: %s\n", version.GetBuildTime())
		fmt.Printf("Git Commit: %s\n", version.GetGitCommit())
		os.Exit(0)
	}

	// Default: if neither flag is set, run both
	if !*enableHTTP && !*enableMCP {
		*enableHTTP = true
		*enableMCP = true
	}

	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo}),
	)
	slog.SetDefault(logger)

	cfg := config.NewServerConfig()
	registry := tools.NewToolRegistry()
	var mcpServer *server.MCPServer
	var httpServer *server.HTTPServer

	if *enableMCP {
		mcpServer = server.NewMCPServer(registry, logger)
	}
	if *enableHTTP {
		//  Passing an mcp server to HTTP server for tool access
		if mcpServer == nil {
			mcpServer = server.NewMCPServer(registry, logger)
		}
		httpServer = server.NewHTTPServer(mcpServer, cfg.HTTPPort, logger)
	}

	ctx := context.Background()

	// Start servers based on flags
	switch {
	case *enableHTTP && *enableMCP:
		// Both servers: use combined server
		srv := server.NewServer(cfg, mcpServer, httpServer)
		if err := srv.Start(ctx); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case *enableHTTP:
		logger.Info("Starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.Start(); err != nil {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	case *enableMCP:
		if err := mcpServer.Start(ctx); err != nil {
			logger.Error("MCP server error", "error", err)
			os.Exit(1)
		}
	}
}
