package server

import (
	"log/slog"
	"os"
	"testing"

	"mcp-tools-server/pkg/tools"
)

func TestNewMCPServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	toolService, err := NewToolService(registry, logger)
	if err != nil {
		t.Fatalf("Failed to create tool service: %v", err)
	}

	mcpServer := NewMCPServer(toolService, logger)

	if mcpServer == nil {
		t.Fatal("NewMCPServer returned nil")
	}

	if mcpServer.logger != logger {
		t.Error("MCP server does not have correct logger reference")
	}

	if mcpServer.processor == nil {
		t.Error("MCP server did not initialize the JSONRPCProcessor")
	}
}
