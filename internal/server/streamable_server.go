package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"mcp-tools-server/internal/config"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StreamableHTTPServer wraps an MCP SDK server and the HTTP server that serves it.
type StreamableHTTPServer struct {
	logger    *slog.Logger
	server    *http.Server
	port      int
	mcpServer *mcp.Server
	// activeSessions stores session IDs that have been initialized since server start.
	// This is a best-effort view and currently only records sessions when the
	// SDK calls the InitializedHandler. Sessions are not removed automatically
	// here when a session ends; the SDK may provide hooks for that in the
	// future.
	activeSessions map[string]time.Time
	sessionsMu     sync.Mutex
}

// NewStreamableHTTPServer creates a new server using the MCP SDK's streamable handler.
func NewStreamableHTTPServer(cfg *config.ServerConfig, toolService *ToolService, logger *slog.Logger) *StreamableHTTPServer {
	// Create an MCP server
	impl := &mcp.Implementation{Name: "mcp-tools-server", Version: "1.0.0"}

	// Provide ServerOptions to enable KeepAlive and a secure session id generator.
	keepAlive := time.Duration(cfg.StreamableKeepAliveSeconds) * time.Second
	opts := &mcp.ServerOptions{
		// Generate a random session id. The SDK will call this when a new
		// session needs an id. Using crypto/rand for secure random bytes.
		GetSessionID: func() string {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err != nil {
				// fall back to timestamp-based id on error
				return fmt.Sprintf("sid-%d", time.Now().UnixNano())
			}
			return hex.EncodeToString(b)
		},
		// Keep sessions alive by pinging clients. Tuned from config.
		KeepAlive: keepAlive,
		// Attach a logging handler when the client sends notifications/initialized.
		InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
			// Create a session-specific logger backed by the SDK LoggingHandler.
			// Provide a LoggerName so clients can identify the source of logs.
			if req != nil && req.Session != nil {
				lhOpts := &mcp.LoggingHandlerOptions{LoggerName: "mcp-tools-server"}
				sessionLogger := slog.New(mcp.NewLoggingHandler(req.Session, lhOpts))
				sessionLogger.Info("session initialized", "session", req.Session.ID())
			}
			// Record the session id in our in-memory map for admin visibility.
			// This is intentionally simple and thread-safe.
			// Use a short timestamp so operators can inspect recent sessions.
			// If req.Session or req.Session.ID were ever nil, guard defensively.
			if req != nil && req.Session != nil {
				// note: SDK session IDs are expected to be non-empty strings
				sid := req.Session.ID()
				if sid != "" {
					// We don't want to import heavy time packages here; record now.
					// Use the server-level map guarded by mutex; initialize below.
					// We'll store the current time for diagnostic purposes.
					// The sessions map is on the StreamableHTTPServer; we will set it
					// after creating the mcpServer because this closure runs later.
					// To avoid a race on s being nil here, callers that instantiate
					// the server will have the s.activeSessions map set.
					// We cannot reference 's' in this scope, so the caller will wrap
					// this in a small helper below when wiring the ServerOptions.
				}
			}
		},
	}

	mcpServer := mcp.NewServer(impl, opts)

	// Register tools from the existing ToolService into the mcp.Server
	toolService.RegisterTool(mcpServer)

	srv := &StreamableHTTPServer{
		logger:         logger,
		port:           cfg.StreamableHTTPPort,
		mcpServer:      mcpServer,
		activeSessions: make(map[string]time.Time),
	}

	// Re-wire the InitializedHandler to capture session IDs into our struct.
	// The SDK already stores the handler in opts; we set a wrapper that calls
	// the original behavior and also records the session id into srv.activeSessions.
	originalInit := opts.InitializedHandler
	opts.InitializedHandler = func(ctx context.Context, req *mcp.InitializedRequest) {
		if originalInit != nil {
			originalInit(ctx, req)
		}
		if req != nil && req.Session != nil {
			sid := req.Session.ID()
			if sid != "" {
				srv.sessionsMu.Lock()
				srv.activeSessions[sid] = time.Now().UTC()
				srv.sessionsMu.Unlock()
			}
		}
	}

	return srv
}

// Start runs the HTTP server and mounts the SDK's StreamableHTTPHandler at /mcp
func (s *StreamableHTTPServer) Start() error {
	s.logger.Info("Starting Streamable HTTP MCP server", "port", s.port)

	mux := http.NewServeMux()
	// Use the SDK's default StreamableHTTPOptions (stateful). The SDK will
	// create a MemoryEventStore by default when needed. We also attach a
	// logging handler at the session level via the SDK where consumers can
	// use slog.New(mcp.NewLoggingHandler(ss, nil)). For HTTP we don't need to
	// alter the handler options here.
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return s.mcpServer }, nil)
	mux.Handle("/mcp", handler)

	// Admin endpoint to inspect active sessions seen by this server instance.
	mux.HandleFunc("/admin/sessions", s.handleAdminSessions)

	// Configure sensible HTTP server timeouts to prevent indefinitely hung
	// connections (SSE consumers that never close, etc.). These are conservative
	// defaults and can be tuned via config later.
	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       5 * time.Minute,
	}

	return s.server.ListenAndServe()
}

// Stop shuts down the HTTP server and any running MCP sessions.
func (s *StreamableHTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Streamable HTTP MCP server")
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// handleAdminSessions returns a JSON array of active sessions recorded by this server.
// This is a lightweight diagnostic endpoint intended for operators.
func (s *StreamableHTTPServer) handleAdminSessions(w http.ResponseWriter, r *http.Request) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	type sess struct {
		ID        string    `json:"id"`
		SeenAtUTC time.Time `json:"seenAtUtc"`
	}

	list := make([]sess, 0, len(s.activeSessions))
	for id, ts := range s.activeSessions {
		list = append(list, sess{ID: id, SeenAtUTC: ts})
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(list); err != nil {
		http.Error(w, "failed to encode sessions", http.StatusInternalServerError)
		return
	}
}
