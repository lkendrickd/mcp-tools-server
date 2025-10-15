package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"mcp-tools-server/internal/config"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StreamableHTTPServer wraps an MCP SDK server and the HTTP server that serves it.
type StreamableHTTPServer struct {
	logger    *slog.Logger
	server    *http.Server
	port      int
	mcpServer *mcp.Server
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
			lhOpts := &mcp.LoggingHandlerOptions{LoggerName: "mcp-tools-server"}
			sessionLogger := slog.New(mcp.NewLoggingHandler(req.Session, lhOpts))
			sessionLogger.Info("session initialized", "session", req.Session.ID)
		},
	}

	mcpServer := mcp.NewServer(impl, opts)

	// Register tools from the existing ToolService into the mcp.Server
	for _, t := range toolService.GetTools() {
		tool := t
		// Add tool using the SDK convenience. Use the generic handler form which
		// expects (ctx, req, in) -> (result, out, error). We'll accept any input
		// and forward to the existing Tool.Execute.
		mcp.AddTool(mcpServer, &mcp.Tool{Name: tool.Name(), Description: tool.Description()}, func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
			// Try to coerce incoming parameters to a map[string]interface{}
			conv := make(map[string]interface{})
			if m, ok := in.(map[string]any); ok {
				for k, v := range m {
					conv[k] = v
				}
			} else if m2, ok := in.(map[string]interface{}); ok {
				conv = m2
			}
			out, err := tool.Execute(conv)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{}, out, nil
		})
	}

	return &StreamableHTTPServer{
		logger:    logger,
		port:      cfg.StreamableHTTPPort,
		mcpServer: mcpServer,
	}
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

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
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
