package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"mcp-tools-server/internal/config"
	"mcp-tools-server/pkg/tools"
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

	// Create the handler and server instance
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", server.handleMCP)
	httpServer := &http.Server{Handler: mux}

	// Start server in a goroutine
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("Server failed: %v", err)
		}
	}()
	defer httpServer.Shutdown(context.Background())

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

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
		// ... (rest of the assertions)
	})

	t.Run("GET request for SSE stream", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseURL+"/mcp", nil)
		if err != nil {
			t.Fatalf("Failed to create SSE request: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send SSE request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200 for SSE, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
			t.Fatalf("Expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
		}

		eventChan := make(chan string)
		go func() {
			reader := bufio.NewReader(resp.Body)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF && !strings.Contains(err.Error(), "context canceled") {
						t.Logf("Error reading SSE stream: %v", err)
					}
					close(eventChan)
					return
				}
				if strings.HasPrefix(line, "data:") {
					eventChan <- strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				}
			}
		}()

		time.Sleep(50 * time.Millisecond) // Give client time to connect
		broadcastMsg := `{"message":"hello"}`
		server.sseManager.Broadcast([]byte(broadcastMsg))

		select {
		case event := <-eventChan:
			if event != broadcastMsg {
				t.Errorf("Expected SSE data '%s', got '%s'", broadcastMsg, event)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Timed out waiting for SSE event")
		}
	})
}
