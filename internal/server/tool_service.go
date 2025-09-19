package server

import (
	"fmt"
	"log/slog"

	"mcp-tools-server/pkg/tools"
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

	return tool.Execute(args)
}

// GetTools returns the map of tools
func (s *ToolService) GetTools() map[string]tools.Tool {
	return s.tools
}
