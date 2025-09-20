package server

import "mcp-tools-server/pkg/tools"

// MockTool is a helper for testing that implements the tools.Tool interface.
type MockTool struct {
	name        string
	description string
	executeFunc func(args map[string]interface{}) (map[string]interface{}, error)
}

func (m *MockTool) Name() string { return m.name }

func (m *MockTool) Description() string { return m.description }

func (m *MockTool) Execute(args map[string]interface{}) (map[string]interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(args)
	}
	return map[string]interface{}{"success": true}, nil
}

// Ensure MockTool implements the interface.
var _ tools.Tool = &MockTool{}
