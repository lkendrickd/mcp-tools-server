package tools

import (
	"log/slog"
	"os"
	"reflect"
	"testing"
)

// MockTool is a test tool implementation
type MockTool struct {
	name        string
	description string
	executeFunc func(args map[string]interface{}) (map[string]interface{}, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Execute(args map[string]interface{}) (map[string]interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(args)
	}
	return map[string]interface{}{"result": "mock"}, nil
}

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	if registry == nil {
		t.Fatal("NewToolRegistry returned nil")
	}

	if registry.builders == nil {
		t.Fatal("Registry builders map is nil")
	}

	// Check that built-in tools are registered
	available := registry.ListAvailable()
	if len(available) == 0 {
		t.Error("No built-in tools were registered")
	}

	// Check that uuid_gen is registered
	found := false
	for _, name := range available {
		if name == "uuid_gen" {
			found = true
			break
		}
	}
	if !found {
		t.Error("uuid_gen tool was not auto-registered")
	}
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	// Register a mock tool
	registry.Register("test_tool", func(logger *slog.Logger, config map[string]string) (Tool, error) {
		return &MockTool{
			name:        "test_tool",
			description: "A test tool",
		}, nil
	})

	available := registry.ListAvailable()
	found := false
	for _, name := range available {
		if name == "test_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Registered tool not found in available tools list")
	}
}

func TestToolRegistry_CreateAllAvailable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	registry := NewToolRegistry()

	tools, err := registry.CreateAllAvailable(logger)
	if err != nil {
		t.Fatalf("CreateAllAvailable failed: %v", err)
	}

	if len(tools) == 0 {
		t.Error("No tools were created")
	}

	// Check that at least the UUID generator was created
	found := false
	for _, tool := range tools {
		if tool.Name() == "generate_uuid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("UUID generator tool was not created")
	}
}

func TestToolRegistry_CreateSpecific(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	registry := NewToolRegistry()

	// Add a test tool
	registry.Register("test_tool", func(logger *slog.Logger, config map[string]string) (Tool, error) {
		return &MockTool{
			name:        "test_tool",
			description: "A test tool",
		}, nil
	})

	t.Run("creates specified tools", func(t *testing.T) {
		tools, err := registry.CreateSpecific(logger, []string{"test_tool"})
		if err != nil {
			t.Fatalf("CreateSpecific failed: %v", err)
		}

		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}

		if tools[0].Name() != "test_tool" {
			t.Errorf("Expected tool name 'test_tool', got '%s'", tools[0].Name())
		}
	})

	t.Run("fails for unknown tool", func(t *testing.T) {
		_, err := registry.CreateSpecific(logger, []string{"unknown_tool"})
		if err == nil {
			t.Error("Expected error for unknown tool, got nil")
		}
	})
}

func TestToolRegistry_ListAvailable(t *testing.T) {
	registry := NewToolRegistry()

	// Register test tools
	registry.Register("tool1", func(logger *slog.Logger, config map[string]string) (Tool, error) {
		return &MockTool{name: "tool1"}, nil
	})
	registry.Register("tool2", func(logger *slog.Logger, config map[string]string) (Tool, error) {
		return &MockTool{name: "tool2"}, nil
	})

	available := registry.ListAvailable()

	// Should have at least uuid_gen + our test tools
	if len(available) < 3 {
		t.Errorf("Expected at least 3 available tools, got %d", len(available))
	}

	// Check that our test tools are listed
	expectedTools := []string{"tool1", "tool2", "uuid_gen"}
	for _, expected := range expectedTools {
		found := false
		for _, actual := range available {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool '%s' not found in available tools", expected)
		}
	}
}

func TestToolRegistry_getEnvironmentConfig(t *testing.T) {
	registry := NewToolRegistry()

	// Set a test environment variable
	testKey := "TEST_TOOL_CONFIG"
	testValue := "test_value"
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	config := registry.getEnvironmentConfig()

	if config[testKey] != testValue {
		t.Errorf("Expected config[%s] = %s, got %s", testKey, testValue, config[testKey])
	}
}

func TestToolInterface(t *testing.T) {
	// Test that our mock tool properly implements the Tool interface
	var _ Tool = &MockTool{}

	mockTool := &MockTool{
		name:        "test",
		description: "test description",
		executeFunc: func(args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"test": "result"}, nil
		},
	}

	if mockTool.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", mockTool.Name())
	}

	if mockTool.Description() != "test description" {
		t.Errorf("Expected description 'test description', got '%s'", mockTool.Description())
	}

	result, err := mockTool.Execute(nil)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	expected := map[string]interface{}{"test": "result"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %v, got %v", expected, result)
	}
}
