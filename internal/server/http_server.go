package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"mcp-tools-server/internal/version"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"code", "method", "endpoint"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "method", "endpoint"},
	)
)

// HTTPServer handles HTTP API requests
type HTTPServer struct {
	toolService *ToolService
	port        int
	server      *http.Server
	logger      *slog.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(toolService *ToolService, port int, logger *slog.Logger) *HTTPServer {
	mux := http.NewServeMux()
	httpServer := &HTTPServer{
		toolService: toolService,
		port:        port,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		logger: logger,
	}

	if err := prometheus.Register(requestsTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}
	if err := prometheus.Register(requestDuration); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}

	// Create API subrouter
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/uuid", httpServer.instrumentHandler("uuid", httpServer.handleUUID))
	apiMux.HandleFunc("/list", httpServer.instrumentHandler("list", httpServer.handleList))
	apiMux.Handle("/metrics", promhttp.Handler())

	// Mount API subrouter under /api/
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))

	// Register other routes
	mux.HandleFunc("/health", httpServer.handleHealth)
	mux.HandleFunc("/", httpServer.handleIndex)

	return httpServer
}

// instrumentHandler wraps a handler with Prometheus metrics instrumentation
func (s *HTTPServer) instrumentHandler(endpoint string, handler http.HandlerFunc) http.HandlerFunc {
	return promhttp.InstrumentHandlerDuration(
		requestDuration.MustCurryWith(prometheus.Labels{"endpoint": endpoint}),
		promhttp.InstrumentHandlerCounter(
			requestsTotal.MustCurryWith(prometheus.Labels{"endpoint": endpoint}),
			handler,
		),
	)
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

	result, err := s.toolService.ExecuteTool("generate_uuid", nil)
	if err != nil {
		s.logger.Error("Failed to execute generate_uuid tool", "error", err)
		http.Error(w, "Failed to generate UUID", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"uuid": result["uuid"].(string),
	}); err != nil {
		s.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleList handles GET /api/list requests
func (s *HTTPServer) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.logger.Warn("Method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.toolService.ListTools()); err != nil {
		s.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	}); err != nil {
		s.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
