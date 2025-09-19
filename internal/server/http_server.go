package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"mcp-tools-server/internal/version"
)

// HTTPServer handles HTTP API requests
type HTTPServer struct {
	mcpServer *MCPServer
	port      int
	server    *http.Server
	logger    *slog.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(mcpServer *MCPServer, port int, logger *slog.Logger) *HTTPServer {
	mux := http.NewServeMux()
	httpServer := &HTTPServer{
		mcpServer: mcpServer,
		port:      port,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		logger: logger,
	}

	// Create API subrouter
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/uuid", httpServer.handleUUID)
	apiMux.HandleFunc("/list", httpServer.handleList)

	// Mount API subrouter under /api/
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))

	// Register other routes
	mux.HandleFunc("/health", httpServer.handleHealth)
	mux.HandleFunc("/", httpServer.handleIndex)

	return httpServer
}

// Start begins the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server", "port", s.port)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	return s.server.Shutdown(ctx)
}

// handleUUID handles GET /api/uuid requests
func (s *HTTPServer) handleUUID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.logger.Warn("Method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get UUID generator from MCP server tools
	uuidGen, exists := s.mcpServer.Tools["generate_uuid"]
	if !exists {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "UUID generator not available",
		})
		return
	}

	uuid, err := uuidGen.Execute(nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to generate UUID",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"uuid": uuid["uuid"].(string),
	})
}

// handleList handles GET /api/list requests
func (s *HTTPServer) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.logger.Warn("Method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create a map of tool names to descriptions
	tools := make(map[string]string)
	for _, tool := range s.mcpServer.Tools {
		tools[tool.Name()] = tool.Description()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tools)
}

// handleHealth handles GET /health requests
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.logger.Warn("Method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// handleIndex handles GET / requests
func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.logger.Warn("Method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"service":   "MCP Tools Server",
		"version":   version.GetVersion(),
		"buildTime": version.GetBuildTime(),
		"gitCommit": version.GetGitCommit(),
		"message":   "Welcome to Go MCP Tools Server!",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
