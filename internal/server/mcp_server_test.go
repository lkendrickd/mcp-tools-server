package server

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"mcp-tools-server/pkg/tools"
)

func TestNewMCPServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()

	mcpServer := NewMCPServer(registry, logger)

	if mcpServer == nil {
		t.Fatal("NewMCPServer returned nil")
	}

	if mcpServer.logger != logger {
		t.Error("MCP server does not have correct logger reference")
	}

	if mcpServer.Tools == nil {
		t.Error("MCP server Tools map is nil")
	}

	// Check that tools were loaded
	if len(mcpServer.Tools) == 0 {
		t.Error("Expected at least one tool to be loaded")
	}

	// Check that UUID generator is available
	if _, exists := mcpServer.Tools["generate_uuid"]; !exists {
		t.Error("Expected 'generate_uuid' tool to be available")
	}
}

func TestNewMCPServer_WithFailingRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Create a registry that will fail to create tools
	registry := &tools.ToolRegistry{}

	mcpServer := NewMCPServer(registry, logger)

	if mcpServer == nil {
		t.Fatal("NewMCPServer returned nil")
	}

	// Should have empty tools map when registry fails
	if len(mcpServer.Tools) != 0 {
		t.Errorf("Expected empty tools map when registry fails, got %d tools", len(mcpServer.Tools))
	}
}

func TestMCPServer_getAvailableTools(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	availableTools := mcpServer.getAvailableTools()

	if len(availableTools) == 0 {
		t.Error("Expected at least one available tool")
	}

	// Check structure of tool info
	for _, toolInfo := range availableTools {
		name, nameOk := toolInfo["name"].(string)
		description, descOk := toolInfo["description"].(string)
		inputSchema, schemaOk := toolInfo["inputSchema"]

		if !nameOk || name == "" {
			t.Error("Tool info has invalid or empty name")
		}
		if !descOk || description == "" {
			t.Error("Tool info has invalid or empty description")
		}
		if !schemaOk || inputSchema == nil {
			t.Error("Tool info has nil InputSchema")
		}
	}

	// Check that UUID generator is in the list
	found := false
	for _, toolInfo := range availableTools {
		if name, ok := toolInfo["name"].(string); ok && name == "generate_uuid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'generate_uuid' tool in available tools")
	}
}

func TestMCPServer_handleMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	t.Run("handles initialized notification", func(t *testing.T) {
		message := map[string]interface{}{
			"method": "initialized",
		}

		err := mcpServer.handleMessage(message)
		if err != nil {
			t.Errorf("handleMessage failed for initialized: %v", err)
		}
	})

	t.Run("handles unknown method without id", func(t *testing.T) {
		message := map[string]interface{}{
			"method": "unknown_method",
		}

		err := mcpServer.handleMessage(message)
		if err != nil {
			t.Errorf("handleMessage should not fail for unknown notification: %v", err)
		}
	})

	t.Run("fails for message without method", func(t *testing.T) {
		message := map[string]interface{}{
			"id": 1,
		}

		err := mcpServer.handleMessage(message)
		if err == nil {
			t.Error("Expected error for message without method")
		}
	})
}

func TestMCPServer_ToolsMap(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	// Test that tools are properly accessible via the Tools map
	for name, tool := range mcpServer.Tools {
		if name == "" {
			t.Error("Empty tool name in Tools map")
		}

		if tool == nil {
			t.Errorf("Nil tool for name '%s'", name)
		}

		// Test that the tool implements the Tool interface properly
		if tool.Name() == "" {
			t.Errorf("Tool '%s' has empty Name()", name)
		}

		if tool.Description() == "" {
			t.Errorf("Tool '%s' has empty Description()", name)
		}

		// Test that Execute method works
		result, err := tool.Execute(map[string]interface{}{})
		if err != nil {
			t.Errorf("Tool '%s' Execute failed: %v", name, err)
		}

		if result == nil {
			t.Errorf("Tool '%s' Execute returned nil result", name)
		}
	}
}

func TestMCPServer_sendError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	// Note: This test doesn't actually capture the output since sendError writes to stdout
	// In a real scenario, you might want to mock os.Stdout or use dependency injection
	err := mcpServer.sendError(1, -32601, "Method not found")
	if err != nil {
		t.Errorf("sendError failed: %v", err)
	}
}

func TestMCPServer_handleToolsList(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	// Note: This test writes to stdout, so we can't easily capture the output
	// But we can test that it doesn't return an error
	err := mcpServer.handleToolsList(42)
	if err != nil {
		t.Errorf("handleToolsList failed: %v", err)
	}
}

func TestMCPServer_handleToolsCall(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	t.Run("successful tool call", func(t *testing.T) {
		message := map[string]interface{}{
			"id": 1,
			"params": map[string]interface{}{
				"name":      "generate_uuid",
				"arguments": map[string]interface{}{},
			},
		}

		// Note: This writes to stdout, but we can test it doesn't error
		err := mcpServer.handleToolsCall(message, 1)
		if err != nil {
			t.Errorf("handleToolsCall failed: %v", err)
		}
	})

	t.Run("missing params", func(t *testing.T) {
		message := map[string]interface{}{
			"id": 1,
		}

		err := mcpServer.handleToolsCall(message, 1)
		if err != nil {
			t.Errorf("handleToolsCall should handle invalid params gracefully: %v", err)
		}
	})

	t.Run("missing tool name", func(t *testing.T) {
		message := map[string]interface{}{
			"id": 1,
			"params": map[string]interface{}{
				"arguments": map[string]interface{}{},
			},
		}

		err := mcpServer.handleToolsCall(message, 1)
		if err != nil {
			t.Errorf("handleToolsCall should handle missing tool name gracefully: %v", err)
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		message := map[string]interface{}{
			"id": 1,
			"params": map[string]interface{}{
				"name":      "nonexistent_tool",
				"arguments": map[string]interface{}{},
			},
		}

		err := mcpServer.handleToolsCall(message, 1)
		if err != nil {
			t.Errorf("handleToolsCall should handle unknown tool gracefully: %v", err)
		}
	})

	t.Run("tool execution error", func(t *testing.T) {
		// Create a mock tool that returns an error
		mockTool := &MockTool{
			name:        "failing_tool",
			description: "A tool that fails",
			executeFunc: func(args map[string]interface{}) (map[string]interface{}, error) {
				return nil, fmt.Errorf("mock execution error")
			},
		}

		// Create MCP server with the failing tool
		mcpServer := &MCPServer{
			Tools: map[string]tools.Tool{
				"failing_tool": mockTool,
			},
			logger: logger,
		}

		message := map[string]interface{}{
			"id": 1,
			"params": map[string]interface{}{
				"name":      "failing_tool",
				"arguments": map[string]interface{}{},
			},
		}

		err := mcpServer.handleToolsCall(message, 1)
		if err != nil {
			t.Errorf("handleToolsCall should handle tool execution error gracefully: %v", err)
		}
	})
}

func TestMCPServer_handleMessage_MoreCases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	mcpServer := NewMCPServer(registry, logger)

	t.Run("handles tools/list request", func(t *testing.T) {
		message := map[string]interface{}{
			"method": "tools/list",
			"id":     1,
		}

		err := mcpServer.handleMessage(message)
		if err != nil {
			t.Errorf("handleMessage failed for tools/list: %v", err)
		}
	})

	t.Run("handles tools/call request", func(t *testing.T) {
		message := map[string]interface{}{
			"method": "tools/call",
			"id":     1,
			"params": map[string]interface{}{
				"name":      "generate_uuid",
				"arguments": map[string]interface{}{},
			},
		}

		err := mcpServer.handleMessage(message)
		if err != nil {
			t.Errorf("handleMessage failed for tools/call: %v", err)
		}
	})

	t.Run("handles unknown method with id", func(t *testing.T) {
		message := map[string]interface{}{
			"method": "unknown_method_with_id",
			"id":     1,
		}

		err := mcpServer.handleMessage(message)
		if err != nil {
			t.Errorf("handleMessage should handle unknown method with id gracefully: %v", err)
		}
	})
}
