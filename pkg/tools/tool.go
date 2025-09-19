package tools

import (
	"fmt"
	"log/slog"
	"os"
)

// Tool is an interface for tools that can be registered with the MCP server. This ensures all tools are uniform.
type Tool interface {
	Name() string
	Description() string
	Execute(args map[string]interface{}) (map[string]interface{}, error)
}

// ToolBuilder is a function that creates a tool with given dependencies
type ToolBuilder func(logger *slog.Logger, config map[string]string) (Tool, error)

// ToolRegistry manages tool creation and discovery
type ToolRegistry struct {
	builders map[string]ToolBuilder
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		builders: make(map[string]ToolBuilder),
	}

	// Auto-register all known tools
	registry.registerBuiltinTools()
	return registry
}

// registerBuiltinTools registers all available tool builders
func (tr *ToolRegistry) registerBuiltinTools() {
	// Register UUID generator (no config needed)
	tr.Register("uuid_gen", func(logger *slog.Logger, config map[string]string) (Tool, error) {
		return NewUUIDGen(logger), nil
	})
}

// Register adds a tool builder to the registry
func (tr *ToolRegistry) Register(name string, builder ToolBuilder) {
	tr.builders[name] = builder
}

// CreateAllAvailable creates all tools that have their dependencies satisfied
func (tr *ToolRegistry) CreateAllAvailable(logger *slog.Logger) ([]Tool, error) {
	// Get all environment variables as config
	config := tr.getEnvironmentConfig()

	var tools []Tool
	var errors []error

	for name, builder := range tr.builders {
		tool, err := builder(logger, config)
		if err != nil {
			logger.Warn("Skipping tool", "tool", name, "reason", err.Error())
			errors = append(errors, fmt.Errorf("tool %s: %w", name, err))
			continue
		}

		tools = append(tools, tool)
		logger.Info("Created tool", "tool", name, "actual_name", tool.Name())
	}

	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools could be created: %v", errors)
	}

	return tools, nil
}

// CreateSpecific creates only the specified tools
func (tr *ToolRegistry) CreateSpecific(logger *slog.Logger, toolNames []string) ([]Tool, error) {
	config := tr.getEnvironmentConfig()

	var tools []Tool
	for _, name := range toolNames {
		builder, exists := tr.builders[name]
		if !exists {
			return nil, fmt.Errorf("unknown tool: %s", name)
		}

		tool, err := builder(logger, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create tool %s: %w", name, err)
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// ListAvailable returns the names of all registered tools
func (tr *ToolRegistry) ListAvailable() []string {
	var names []string
	for name := range tr.builders {
		names = append(names, name)
	}
	return names
}

// getEnvironmentConfig reads all environment variables into a config map
func (tr *ToolRegistry) getEnvironmentConfig() map[string]string {
	config := make(map[string]string)
	for _, env := range os.Environ() {
		// Parse "KEY=value" format
		for i := 0; i < len(env); i++ {
			if env[i] == '=' {
				key := env[:i]
				value := env[i+1:]
				config[key] = value
				break
			}
		}
	}
	return config
}
