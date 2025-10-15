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
		enableWebSocket   = flag.Bool("websocket", false, "Enable WebSocket server")
		enableAll         = flag.Bool("all", false, "Enable all server modes")
		streamablePort    = flag.Int("streamable-port", 0, "Port for Streamable HTTP MCP server (overrides env)")
		httpPort          = flag.Int("http-port", 0, "Port for HTTP REST server (overrides env)")
		webSocketPort     = flag.Int("websocket-port", 0, "Port for WebSocket server (overrides env)")
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
	runWebSocket := *enableWebSocket

	if *enableAll {
		runMCP, runHTTP, runStreamable, runWebSocket = true, true, true, true
	} else if !runMCP && !runHTTP && !runStreamable && !runWebSocket {
		// Default: run all servers if no specific flag is set
		runMCP, runHTTP, runStreamable, runWebSocket = true, true, true, true
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
	if *webSocketPort != 0 {
		cfg.WebSocketPort = *webSocketPort
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
	var webSocketServer *server.WebSocketServer

	if runMCP {
		mcpServer = server.NewMCPServer(cfg, toolService, logger)
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
	if runWebSocket {
		// Ensure we have an MCP SDK server to back the WebSocket server. If the
		// stdio MCP server wasn't enabled explicitly, create an SDK server here
		// so the WebSocket path is always handled by the SDK.
		if mcpServer == nil {
			mcpServer = server.NewMCPServer(cfg, toolService, logger)
		}
		webSocketServer = server.NewWebSocketServer(cfg, mcpServer.Server())
		logger.Info("WebSocket server enabled (SDK-backed)", "port", cfg.WebSocketPort)
	}

	// --- Server Start ---
	// The combined server handles the lifecycle of all non-nil servers.
	srv := server.NewServer(cfg, mcpServer, httpServer, streamableHTTPServer, webSocketServer)
	if err := srv.Start(context.Background()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
