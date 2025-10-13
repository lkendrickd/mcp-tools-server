package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mcp-tools-server/internal/config"
	"net/http"
	"time"
)

// StreamableHTTPServer handles the streamable HTTP transport for MCP.
type StreamableHTTPServer struct {
	logger          *slog.Logger
	processor       *JSONRPCProcessor
	sseManager      *SSEManager
	securityManager *SecurityManager
	server          *http.Server
	port            int
}

// NewStreamableHTTPServer creates a new server for the streamable HTTP transport.
func NewStreamableHTTPServer(cfg *config.ServerConfig, toolService *ToolService, logger *slog.Logger) *StreamableHTTPServer {
	processor := NewJSONRPCProcessor(toolService, logger)
	sseManager := NewSSEManager(logger)
	securityManager := NewSecurityManager(cfg.AllowedOrigins, cfg.EnableOriginCheck, logger)

	return &StreamableHTTPServer{
		port:            cfg.StreamableHTTPPort,
		logger:          logger,
		processor:       processor,
		sseManager:      sseManager,
		securityManager: securityManager,
	}
}

// Start runs the streamable HTTP server.
func (s *StreamableHTTPServer) Start() error {
	s.logger.Info("Starting Streamable HTTP MCP server", "port", s.port)
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)

	// Apply security middleware
	handler := s.securityManager.OriginCheckMiddleware(mux)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: handler,
	}

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("streamable http server failed: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the server.
func (s *StreamableHTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Streamable HTTP MCP server")
	if s.server == nil {
		return nil // Server was never started
	}
	// Add a timeout to the context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// handleMCP is the single endpoint for all MCP communication.
func (s *StreamableHTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("Received request for /mcp", "method", r.Method, "remoteAddr", r.RemoteAddr)

	switch r.Method {
	case http.MethodGet:
		s.handleSSEConnection(w, r)
	case http.MethodPost:
		s.handlePostRequest(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePostRequest handles incoming JSON-RPC messages via POST.
func (s *StreamableHTTPServer) handlePostRequest(w http.ResponseWriter, r *http.Request) {
	// Validate headers
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Decode the incoming JSON-RPC message
	var message map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "Failed to decode JSON body", http.StatusBadRequest)
		return
	}

	method, ok := message["method"].(string)
	if !ok {
		http.Error(w, "Invalid JSON-RPC: missing method", http.StatusBadRequest)
		return
	}

	id, hasId := message["id"]
	var response *JSONRPCResponse

	// Process the message
	switch method {
	case "initialize":
		if !hasId {
			http.Error(w, "Invalid initialize: missing id", http.StatusBadRequest)
			return
		}
		response = s.processor.HandleInitialize(id)
	case "initialized":
		// This is a notification, respond with 202 Accepted
		w.WriteHeader(http.StatusAccepted)
		return
	case "tools/list":
		if !hasId {
			http.Error(w, "Invalid tools/list: missing id", http.StatusBadRequest)
			return
		}
		response = s.processor.HandleToolsList(id)
	case "tools/call":
		if !hasId {
			http.Error(w, "Invalid tools/call: missing id", http.StatusBadRequest)
			return
		}
		params, _ := message["params"].(map[string]interface{})
		response = s.processor.HandleToolsCall(params, id)
	default:
		if hasId {
			response = s.processor.CreateErrorResponse(id, -32601, "Method not found")
		} else {
			// It's an unknown notification, just accept it
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}

	// For now, always send an immediate JSON response.
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(response); err != nil {
		s.logger.Error("Failed to encode and send response", "error", err)
		http.Error(w, "Failed to send response", http.StatusInternalServerError)
		return
	}

	// Also broadcast the JSON-RPC response to any connected SSE clients so
	// GET /mcp listeners can receive server-generated messages (streaming).
	if s.sseManager != nil && response != nil {
		if b, err := json.Marshal(response); err == nil {
			s.sseManager.Broadcast(b)
		} else {
			s.logger.Warn("Failed to marshal response for SSE broadcast", "error", err)
		}
	}
}

// handleSSEConnection handles a new client connection for receiving server-sent events.
func (s *StreamableHTTPServer) handleSSEConnection(w http.ResponseWriter, r *http.Request) {
	// Check for SSE support
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush() // Immediately send headers

	// Add client to the manager
	client := s.sseManager.AddClient()
	defer s.sseManager.RemoveClient(client.id)

	s.logger.Info("SSE client connected", "clientID", client.id)

	// Keep connection alive and listen for messages
	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				// Channel was closed, client is being removed.
				s.logger.Info("SSE channel closed for client", "clientID", client.id)
				return
			}
			// Format as SSE message (data: <message>\n\n)
			fmt.Fprintf(w, "data: %s\n\n", message)
			flusher.Flush()
		case <-r.Context().Done():
			// Client has disconnected
			s.logger.Info("SSE client disconnected", "clientID", client.id)
			return
		}
	}
}
