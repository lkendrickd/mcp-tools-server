package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"mcp-tools-server/pkg/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolService handles the creation and execution of tools
type ToolService struct {
	tools  map[string]tools.Tool
	logger *slog.Logger
}

// NewToolService creates a new ToolService
func NewToolService(registry *tools.ToolRegistry, logger *slog.Logger) (*ToolService, error) {
	service := &ToolService{
		tools:  make(map[string]tools.Tool),
		logger: logger,
	}

	availableTools, err := registry.CreateAllAvailable(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create tools from registry: %w", err)
	}

	for _, tool := range availableTools {
		service.tools[tool.Name()] = tool
	}

	logger.Info("Registered tools", "count", len(service.tools))
	return service, nil
}

// ListTools returns a map of tool names to their descriptions
func (s *ToolService) ListTools() map[string]string {
	toolList := make(map[string]string)
	for name, tool := range s.tools {
		toolList[name] = tool.Description()
	}
	return toolList
}

// ExecuteTool executes a tool with the given name and arguments
func (s *ToolService) ExecuteTool(name string, args map[string]interface{}) (map[string]interface{}, error) {
	tool, exists := s.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	result, err := tool.Execute(args)
	if err != nil {
		s.logger.Error("Tool execution failed", "tool", name, "error", err)
		return nil, err
	}

	// Log the result for cross-verification
	s.logger.Info("Tool executed successfully", "tool", name, "result", result)

	return result, nil
}

// GetTools returns the map of tools
func (s *ToolService) GetTools() map[string]tools.Tool {
	return s.tools
}

// RegisterTool registers all known tools onto the provided SDK server.
// This centralizes the translation between internal Tool and the SDK's mcp.Tool
// and ensures a single place to modify behavior when adapting inputs/outputs.
func (s *ToolService) RegisterTool(srv *mcp.Server) {
	for _, t := range s.tools {
		tool := t
		mcp.AddTool(srv, &mcp.Tool{Name: tool.Name(), Description: tool.Description()}, func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
			conv := make(map[string]interface{})
			if m, ok := in.(map[string]any); ok {
				for k, v := range m {
					conv[k] = v
				}
			} else if m2, ok := in.(map[string]interface{}); ok {
				conv = m2
			}
			out, err := tool.Execute(conv)
			if err != nil {
				return nil, nil, err
			}
			norm, err := normalizeToolResult(out)
			if err != nil {
				// If normalization fails, return the original output as a string fallback
				return nil, nil, fmt.Errorf("failed to normalize tool result for %s: %w", tool.Name(), err)
			}
			return &mcp.CallToolResult{}, norm, nil
		})
	}
}

// normalizeToolResult coerces arbitrary tool outputs into a predictable
// map[string]interface{} shape that callers and clients can rely on.
func normalizeToolResult(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return map[string]interface{}{}, nil
	}

	// Fast path: already the desired map shape
	if m, ok := v.(map[string]interface{}); ok {
		// Prefer structuredContent if present
		if sc, exists := m["structuredContent"]; exists {
			if sm, ok := sc.(map[string]interface{}); ok {
				return sm, nil
			}
			return map[string]interface{}{"items": sc}, nil
		}
		return m, nil
	}

	// Accept map[string]any as well
	if m2, ok := v.(map[string]any); ok {
		out := make(map[string]interface{}, len(m2))
		for k, val := range m2 {
			out[k] = val
		}
		if sc, exists := out["structuredContent"]; exists {
			if sm, ok := sc.(map[string]interface{}); ok {
				return sm, nil
			}
			return map[string]interface{}{"items": sc}, nil
		}
		return out, nil
	}

	// If it's a JSON string or bytes try to unmarshal
	switch b := v.(type) {
	case []byte:
		var mm map[string]interface{}
		if err := json.Unmarshal(b, &mm); err == nil {
			return normalizeToolResult(mm)
		}
		var aa []interface{}
		if err := json.Unmarshal(b, &aa); err == nil {
			return map[string]interface{}{"items": aa}, nil
		}
	case string:
		return normalizeToolResult([]byte(b))
	}

	// Fallback: marshal then unmarshal to map to handle structs
	j, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal fallback failed: %w", err)
	}
	var mm map[string]interface{}
	if err := json.Unmarshal(j, &mm); err == nil {
		return normalizeToolResult(mm)
	}
	var aa []interface{}
	if err := json.Unmarshal(j, &aa); err == nil {
		return map[string]interface{}{"items": aa}, nil
	}

	// As a last resort, return the JSON string under `value`
	return map[string]interface{}{"value": string(j)}, nil
}
