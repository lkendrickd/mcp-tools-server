package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"mcp-tools-server/internal/config"
	"mcp-tools-server/internal/server"
	"mcp-tools-server/internal/version"
	"mcp-tools-server/pkg/tools"
)

func main() {
	// --- Flag Definition ---
	var (
		showVersion       = flag.Bool("version", false, "Show version and exit")
		enableHTTP        = flag.Bool("http", false, "Enable HTTP REST server")
		enableMCP         = flag.Bool("mcp", false, "Enable stdio MCP server")
		enableStreamable  = flag.Bool("streamable", false, "Enable Streamable HTTP MCP server")
		enableAll         = flag.Bool("all", false, "Enable all three server modes")
		streamablePort    = flag.Int("streamable-port", 0, "Port for Streamable HTTP MCP server (overrides env)")
		httpPort          = flag.Int("http-port", 0, "Port for HTTP REST server (overrides env)")
		enableOriginCheck = flag.Bool("enable-origin-check", false, "Enable origin check for streamable server")
		allowedOriginsRaw = flag.String("allowed-origins", "", "Comma-separated list of allowed origins (overrides env)")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("MCP Tools Server v%s\n", version.GetVersion())
		os.Exit(0)
	}

	// --- Server Mode Logic ---
	runMCP := *enableMCP
	runHTTP := *enableHTTP
	runStreamable := *enableStreamable

	if *enableAll {
		runMCP, runHTTP, runStreamable = true, true, true
	} else if !runMCP && !runHTTP && !runStreamable {
		// Default: run all servers if no specific flag is set
		runMCP, runHTTP, runStreamable = true, true, true
	}

	// --- Configuration Loading ---
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.NewServerConfig()
	// Override config with flags if they were provided
	if *httpPort != 0 {
		cfg.HTTPPort = *httpPort
	}
	if *streamablePort != 0 {
		cfg.StreamableHTTPPort = *streamablePort
	}
	// For bool flags, we need to check if the flag was actually set on the command line
	// to differentiate it from the default `false` value.
	isOriginCheckSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "enable-origin-check" {
			isOriginCheckSet = true
		}
	})
	if isOriginCheckSet {
		cfg.EnableOriginCheck = *enableOriginCheck
	}
	if *allowedOriginsRaw != "" {
		cfg.AllowedOrigins = strings.Split(*allowedOriginsRaw, ",")
	}

	// --- Service and Server Initialization ---
	registry := tools.NewToolRegistry()
	toolService, err := server.NewToolService(registry, logger)
	if err != nil {
		logger.Error("Failed to create tool service", "error", err)
		os.Exit(1)
	}

	var mcpServer *server.MCPServer
	var httpServer *server.HTTPServer
	var streamableHTTPServer *server.StreamableHTTPServer

	if runMCP {
		mcpServer = server.NewMCPServer(toolService, logger)
		logger.Info("Stdio MCP server enabled")
	}
	if runHTTP {
		httpServer = server.NewHTTPServer(toolService, cfg.HTTPPort, logger)
		logger.Info("HTTP REST server enabled", "port", cfg.HTTPPort)
	}
	if runStreamable {
		streamableHTTPServer = server.NewStreamableHTTPServer(cfg, toolService, logger)
		logger.Info("Streamable HTTP MCP server enabled", "port", cfg.StreamableHTTPPort, "origin-check", cfg.EnableOriginCheck)
	}

	// --- Server Start ---
	// The combined server handles the lifecycle of all non-nil servers.
	srv := server.NewServer(cfg, mcpServer, httpServer, streamableHTTPServer)
	if err := srv.Start(context.Background()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
