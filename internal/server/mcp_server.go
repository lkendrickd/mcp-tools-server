package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"mcp-tools-server/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer wraps an mcp.Server and runs it on the StdioTransport.
type MCPServer struct {
	logger *slog.Logger
	srv    *mcp.Server
}

// NewMCPServer creates a new MCPServer backed by the SDK Server, registering
// tools from the provided ToolService.
func NewMCPServer(cfg *config.ServerConfig, toolService *ToolService, logger *slog.Logger) *MCPServer {
	impl := &mcp.Implementation{Name: "mcp-tools-server", Version: "1.0.0"}

	// Use configured keepalive (seconds) to drive SDK KeepAlive.
	keepAlive := time.Duration(cfg.StdioKeepAliveSeconds) * time.Second
	opts := &mcp.ServerOptions{
		GetSessionID: func() string {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err != nil {
				return "sid-stdio"
			}
			return hex.EncodeToString(b)
		},
		KeepAlive: keepAlive,
		InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
			lhOpts := &mcp.LoggingHandlerOptions{LoggerName: "mcp-tools-server-stdio"}
			sessionLogger := slog.New(mcp.NewLoggingHandler(req.Session, lhOpts))
			sessionLogger.Info("stdio session initialized", "session", req.Session.ID)
		},
	}

	srv := mcp.NewServer(impl, opts)

	// Register tools on the SDK server
	toolService.RegisterTool(srv)

	return &MCPServer{logger: logger, srv: srv}
}

// Server returns the underlying SDK server instance.
func (s *MCPServer) Server() *mcp.Server { return s.srv }

// Start launches the SDK server on the standard StdioTransport. Run blocks until
// the transport session ends or ctx is cancelled.
func (s *MCPServer) Start(ctx context.Context) error {
	s.logger.Info("Starting MCP server (stdio)")
	return s.srv.Run(ctx, &mcp.StdioTransport{})
}
