package server

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"mcp-tools-server/pkg/tools"
)

func setupProcessor(t *testing.T) *JSONRPCProcessor {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	toolService, err := NewToolService(registry, logger)
	if err != nil {
		t.Fatalf("Failed to create tool service: %v", err)
	}
	return NewJSONRPCProcessor(toolService, logger)
}

func TestJSONRPCProcessor_HandleInitialize(t *testing.T) {
	p := setupProcessor(t)
	resp := p.HandleInitialize(1)

	if resp.ID != 1 {
		t.Errorf("Expected ID 1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("Expected result, got nil")
	}

	result, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("Unexpected result type: %T", resp.Result)
	}

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("Wrong protocol version: %s", result.ProtocolVersion)
	}
	if len(result.Capabilities) == 0 {
		t.Error("Expected capabilities, got none")
	}
}

func TestJSONRPCProcessor_HandleToolsList(t *testing.T) {
	p := setupProcessor(t)
	resp := p.HandleToolsList(42)

	if resp.ID != 42 {
		t.Errorf("Expected ID 42, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("Expected result, got nil")
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected result type: %T", resp.Result)
	}

	tools, ok := result["tools"].([]ToolDefinition)
	if !ok {
		t.Fatalf("Unexpected tools type: %T", result["tools"])
	}

	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}
}

func TestJSONRPCProcessor_HandleToolsCall(t *testing.T) {
	p := setupProcessor(t)

	t.Run("successful tool call", func(t *testing.T) {
		params := map[string]interface{}{
			"name":      "generate_uuid",
			"arguments": map[string]interface{}{},
		}
		resp := p.HandleToolsCall(params, 1)

		if resp.Error != nil {
			t.Errorf("Expected no error, got %v", resp.Error)
		}
		if resp.Result == nil {
			t.Fatal("Expected result, got nil")
		}
		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("Unexpected result type: %T", resp.Result)
		}
		if _, ok := result["uuid"]; !ok {
			t.Error("Expected uuid in result")
		}
	})

	t.Run("missing tool name", func(t *testing.T) {
		params := map[string]interface{}{"arguments": map[string]interface{}{}}
		resp := p.HandleToolsCall(params, 2)
		if resp.Error == nil {
			t.Fatal("Expected error, got nil")
		}
		if resp.Error.Code != -32602 {
			t.Errorf("Expected code -32602, got %d", resp.Error.Code)
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		params := map[string]interface{}{"name": "nonexistent_tool"}
		resp := p.HandleToolsCall(params, 3)
		if resp.Error == nil {
			t.Fatal("Expected error, got nil")
		}
		if resp.Error.Code != -32000 {
			t.Errorf("Expected code -32000, got %d", resp.Error.Code)
		}
	})

	t.Run("tool execution error", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
		mockTool := &MockTool{
			name:        "failing_tool",
			description: "A tool that always fails",
			executeFunc: func(args map[string]interface{}) (map[string]interface{}, error) {
				return nil, fmt.Errorf("mock execution error")
			},
		}
		failingToolService := &ToolService{
			tools:  map[string]tools.Tool{"failing_tool": mockTool},
			logger: logger,
		}
		pWithFailingTool := NewJSONRPCProcessor(failingToolService, logger)

		params := map[string]interface{}{"name": "failing_tool"}
		resp := pWithFailingTool.HandleToolsCall(params, 4)

		if resp.Error == nil {
			t.Fatal("Expected error, got nil")
		}
		if resp.Error.Code != -32000 {
			t.Errorf("Expected code -32000, got %d", resp.Error.Code)
		}
	})
}

func TestJSONRPCProcessor_CreateErrorResponse(t *testing.T) {
	p := setupProcessor(t)
	resp := p.CreateErrorResponse(10, -32601, "Method not found")

	if resp.ID != 10 {
		t.Errorf("Expected ID 10, got %v", resp.ID)
	}
	if resp.Result != nil {
		t.Errorf("Expected no result, got %v", resp.Result)
	}
	if resp.Error == nil {
		t.Fatal("Expected error, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected code -32601, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("Wrong error message: %s", resp.Error.Message)
	}
}
