package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"mcp-tools-server/internal/config"
	"mcp-tools-server/pkg/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupTestServerWithListener creates a new streamable server and a listener on a random port.
func setupTestServerWithListener(t *testing.T) (*StreamableHTTPServer, net.Listener) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cfg := config.NewServerConfig()

	registry := tools.NewToolRegistry()
	toolService, err := NewToolService(registry, logger)
	if err != nil {
		t.Fatalf("Failed to create tool service: %v", err)
	}

	// Create a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// The server will be configured with the listener's port, but we pass the whole config for other settings
	cfg.StreamableHTTPPort = listener.Addr().(*net.TCPAddr).Port
	server := NewStreamableHTTPServer(cfg, toolService, logger)

	return server, listener
}

func TestStreamableHTTPServer_FullFlow(t *testing.T) {
	server, listener := setupTestServerWithListener(t)
	baseURL := "http://" + listener.Addr().String()

	// Create the handler and server instance using the SDK streamable handler
	mux := http.NewServeMux()
	// Use the SDK default handler options (stateful) in tests to match how
	// real clients will establish a session and SSE stream.
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return server.mcpServer }, nil)
	mux.Handle("/mcp", handler)
	httpServer := &http.Server{Handler: mux}

	// Start server in a goroutine
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("Server failed: %v", err)
		}
	}()
	defer func() { _ = httpServer.Shutdown(context.Background()) }()

	t.Run("POST request for tools/call", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params":  map[string]interface{}{"name": "generate_uuid"},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", baseURL+"/mcp", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
        // The SDK StreamableHTTPHandler expects the Accept header to include both
        // application/json and text/event-stream for POST requests that initiate
        // or interact with a streamable session.
        req.Header.Set("Accept", "application/json, text/event-stream")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
	defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
		// ... (rest of the assertions)
	})

	t.Run("GET request for SSE stream", func(t *testing.T) {
		// Perform an initialize POST to create a session and obtain a session id
		initBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "initialize",
			"params":  map[string]interface{}{},
		}
		initBytes, _ := json.Marshal(initBody)
		initReq, err := http.NewRequest("POST", baseURL+"/mcp", bytes.NewReader(initBytes))
		if err != nil {
			t.Fatalf("Failed to create initialize request: %v", err)
		}
		initReq.Header.Set("Content-Type", "application/json")
		initReq.Header.Set("Accept", "application/json, text/event-stream")
		initResp, err := http.DefaultClient.Do(initReq)
		if err != nil {
			t.Fatalf("Failed to send initialize request: %v", err)
		}
	defer func() { _ = initResp.Body.Close() }()
		sessionID := initResp.Header.Get("Mcp-Session-Id")
		if sessionID == "" {
			t.Fatalf("Expected Mcp-Session-Id header in initialize response")
		}

		req, err := http.NewRequest("GET", baseURL+"/mcp", nil)
		if err != nil {
			t.Fatalf("Failed to create SSE request: %v", err)
		}
		// For SSE/stream connections the handler expects Accept to include
		// text/event-stream so the transport upgrades to an event stream.
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Mcp-Session-Id", sessionID)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send SSE request: %v", err)
		}
	defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200 for SSE, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
			t.Fatalf("Expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
		}
		// Note: SDK handles SSE stream establishment; sending server-initiated
		// messages is exercised elsewhere. Here we only assert the connection.
	})
}
