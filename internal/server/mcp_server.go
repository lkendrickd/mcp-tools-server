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
	for _, t := range toolService.GetTools() {
		tool := t
		mcp.AddTool(srv, &mcp.Tool{Name: tool.Name(), Description: tool.Description()}, func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
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

	return &MCPServer{logger: logger, srv: srv}
}

// Start launches the SDK server on the standard StdioTransport. Run blocks until
// the transport session ends or ctx is cancelled.
func (s *MCPServer) Start(ctx context.Context) error {
	s.logger.Info("Starting MCP server (stdio)")
	return s.srv.Run(ctx, &mcp.StdioTransport{})
}
