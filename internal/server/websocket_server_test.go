package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"nhooyr.io/websocket"

	"mcp-tools-server/internal/config"
	"mcp-tools-server/pkg/tools"
)

// TestWebSocketServer_E2E performs an end-to-end test of the WebSocket server.
func TestWebSocketServer_E2E(t *testing.T) {
	// --- Test Setup ---
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := &config.ServerConfig{
		WebSocketPort: 9999,
	}

	// Create a real tool registry and service
	registry := tools.NewToolRegistry()
	toolService, err := NewToolService(registry, logger)
	if err != nil {
		t.Fatalf("Failed to create tool service: %v", err)
	}

	// Create the JSON-RPC processor
	processor := NewJSONRPCProcessor(toolService, logger)

	// Create and start the WebSocket server in a goroutine
	wsServer := NewWebSocketServer(cfg, processor)
	testServer := httptest.NewServer(http.HandlerFunc(wsServer.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// --- Test Execution ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dial the WebSocket server
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial WebSocket server: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 1. Send "initialize" request
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}
	if err := writeRequest(ctx, conn, initRequest); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// 2. Read and validate "initialize" response
	initResp, err := readResponse(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to read initialize response: %v", err)
	}
	if initResp["id"].(float64) != 1 {
		t.Errorf("Expected response ID 1, got %v", initResp["id"])
	}
	if _, ok := initResp["result"].(map[string]interface{})["capabilities"]; !ok {
		t.Error("Expected 'capabilities' in initialize response")
	}

	// 3. Send "tools/call" request
	callRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "generate_uuid",
		},
	}
	if err := writeRequest(ctx, conn, callRequest); err != nil {
		t.Fatalf("Failed to send tools/call request: %v", err)
	}

	// 4. Read and validate "tools/call" response
	callResp, err := readResponse(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to read tools/call response: %v", err)
	}
	if callResp["id"].(float64) != 2 {
		t.Errorf("Expected response ID 2, got %v", callResp["id"])
	}
	result, ok := callResp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", callResp["result"])
	}
	if _, ok := result["uuid"]; !ok {
		t.Error("Expected 'uuid' in tools/call response")
	}
	t.Logf("Received UUID: %s", result["uuid"])
}

// writeRequest is a helper to send a JSON request to the WebSocket connection.
func writeRequest(ctx context.Context, conn *websocket.Conn, req map[string]interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

// readResponse is a helper to read a JSON response from the WebSocket connection.
func readResponse(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	msgType, data, err := conn.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}
	if msgType != websocket.MessageText {
		return nil, fmt.Errorf("expected text message, got %v", msgType)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return resp, nil
}
