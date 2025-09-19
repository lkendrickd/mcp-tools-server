package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"mcp-tools-server/pkg/tools"
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

func setupTestServer() (*HTTPServer, *ToolService) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	toolService, err := NewToolService(registry, logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to create tool service: %v", err))
	}
	httpServer := NewHTTPServer(toolService, 8080, logger)
	return httpServer, toolService
}

func TestHTTPServer_handleIndex(t *testing.T) {
	httpServer, _ := setupTestServer()

	t.Run("GET request returns JSON with service info", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		httpServer.handleIndex(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		expectedFields := []string{"service", "version", "buildTime", "gitCommit", "message"}
		for _, field := range expectedFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in response", field)
			}
		}

		if response["service"] != "MCP Tools Server" {
			t.Errorf("Expected service 'MCP Tools Server', got %v", response["service"])
		}
	})

	t.Run("POST request returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		w := httptest.NewRecorder()

		httpServer.handleIndex(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

func TestHTTPServer_handleHealth(t *testing.T) {
	httpServer, _ := setupTestServer()

	t.Run("GET request returns healthy status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		httpServer.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %s", response["status"])
		}
	})

	t.Run("POST request returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()

		httpServer.handleHealth(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

func TestHTTPServer_handleUUID(t *testing.T) {
	httpServer, _ := setupTestServer()

	t.Run("GET request generates UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/uuid", nil)
		w := httptest.NewRecorder()

		httpServer.handleUUID(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		uuid, exists := response["uuid"]
		if !exists {
			t.Error("Expected 'uuid' field in response")
		}

		// Basic UUID format validation (should be 36 characters with 4 hyphens)
		if len(uuid) != 36 {
			t.Errorf("Expected UUID length 36, got %d", len(uuid))
		}

		hyphenCount := strings.Count(uuid, "-")
		if hyphenCount != 4 {
			t.Errorf("Expected 4 hyphens in UUID, got %d", hyphenCount)
		}
	})

	t.Run("POST request returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/uuid", nil)
		w := httptest.NewRecorder()

		httpServer.handleUUID(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("handles missing UUID tool", func(t *testing.T) {
		// Create a tool service with no tools
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
		toolService := &ToolService{
			tools:  make(map[string]tools.Tool),
			logger: logger,
		}
		httpServer := NewHTTPServer(toolService, 8080, logger)

		req := httptest.NewRequest("GET", "/api/uuid", nil)
		w := httptest.NewRecorder()

		httpServer.handleUUID(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		body := strings.TrimSpace(w.Body.String())
		expectedError := "Failed to generate UUID"
		if body != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, body)
		}
	})

	t.Run("handles tool execution error", func(t *testing.T) {
		// Create a mock tool that returns an error
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
		mockTool := &MockTool{
			name:        "generate_uuid",
			description: "Failing UUID generator",
			executeFunc: func(args map[string]interface{}) (map[string]interface{}, error) {
				return nil, fmt.Errorf("mock execution error")
			},
		}

		toolService := &ToolService{
			tools: map[string]tools.Tool{
				"generate_uuid": mockTool,
			},
			logger: logger,
		}
		httpServer := NewHTTPServer(toolService, 8080, logger)

		req := httptest.NewRequest("GET", "/api/uuid", nil)
		w := httptest.NewRecorder()

		httpServer.handleUUID(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		body := strings.TrimSpace(w.Body.String())
		expectedError := "Failed to generate UUID"
		if body != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, body)
		}
	})
}

func TestHTTPServer_handleList(t *testing.T) {
	httpServer, _ := setupTestServer()

	t.Run("GET request returns available tools", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/list", nil)
		w := httptest.NewRecorder()

		httpServer.handleList(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have at least the UUID generator
		if len(response) == 0 {
			t.Error("Expected at least one tool in response")
		}

		// Check for UUID generator specifically
		if _, exists := response["generate_uuid"]; !exists {
			t.Error("Expected 'generate_uuid' tool in response")
		}
	})

	t.Run("POST request returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/list", nil)
		w := httptest.NewRecorder()

		httpServer.handleList(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

func TestNewHTTPServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := tools.NewToolRegistry()
	toolService, _ := NewToolService(registry, logger)

	httpServer := NewHTTPServer(toolService, 8080, logger)

	if httpServer == nil {
		t.Fatal("NewHTTPServer returned nil")
	}

	if httpServer.toolService != toolService {
		t.Error("HTTP server does not have correct ToolService reference")
	}

	if httpServer.port != 8080 {
		t.Errorf("Expected port 8080, got %d", httpServer.port)
	}

	if httpServer.logger != logger {
		t.Error("HTTP server does not have correct logger reference")
	}
}

func TestHTTPServer_Routes(t *testing.T) {
	httpServer, _ := setupTestServer()

	// Test that routes are properly configured by making requests
	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{"/", http.StatusOK},
		{"/health", http.StatusOK},
		{"/api/uuid", http.StatusOK},
		{"/api/list", http.StatusOK},
		// Note: The current implementation doesn't have a 404 handler,
		// so unknown routes fall through to the root handler
		{"/nonexistent", http.StatusOK}, // This actually gets handled by the root handler
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()

		httpServer.server.Handler.ServeHTTP(w, req)

		if w.Code != tc.expectedStatus {
			t.Errorf("For path %s, expected status %d, got %d", tc.path, tc.expectedStatus, w.Code)
		}
	}
}
