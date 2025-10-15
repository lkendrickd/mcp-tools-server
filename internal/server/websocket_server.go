package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nhooyr.io/websocket"

	"mcp-tools-server/internal/config"
)

// WebSocketServer handles WebSocket connections.
type WebSocketServer struct {
	config     *config.ServerConfig
	httpServer *http.Server
	sdkServer  *mcp.Server
}

// websocketConn implements mcp.Connection over a nhooyr websocket.Conn
type websocketConn struct {
	conn *websocket.Conn
	sid  string
}

func newWebsocketConn(conn *websocket.Conn, sid string) *websocketConn {
	return &websocketConn{conn: conn, sid: sid}
}

func (w *websocketConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	_, data, err := w.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	msg, err := jsonrpc.DecodeMessage(data)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (w *websocketConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}
	return w.conn.Write(ctx, websocket.MessageText, data)
}

func (w *websocketConn) Close() error {
	return w.conn.Close(websocket.StatusNormalClosure, "")
}

func (w *websocketConn) SessionID() string { return w.sid }

// websocketServerTransport is a per-request transport that upgrades the HTTP
// connection and then is connected into the SDK server.
type websocketServerTransport struct {
	conn *websocket.Conn
	sid  string
}

func (t *websocketServerTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	return newWebsocketConn(t.conn, t.sid), nil
}

// NewWebSocketServer creates a new WebSocket server backed by an SDK server.
func NewWebSocketServer(cfg *config.ServerConfig, sdk *mcp.Server) *WebSocketServer {
	return &WebSocketServer{config: cfg, sdkServer: sdk}
}

// Start initializes and starts the WebSocket server.
func (s *WebSocketServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.httpServer = &http.Server{
		Addr:    s.config.WebSocketAddr(),
		Handler: mux,
	}

	log.Printf("WebSocket server listening on %s", s.config.WebSocketAddr())
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop gracefully shuts down the WebSocket server.
func (s *WebSocketServer) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleWebSocket upgrades HTTP connections to WebSocket connections.
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // TODO: Make this configurable
	})
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	// Do not send an internal-error close automatically — only close with an
	// error status when an error actually occurs. The SDK server may take
	// ownership of the connection and manage closure itself.
	closed := false
	defer func() {
		if !closed {
			// Best-effort close if the handler did not already close the conn.
			_ = conn.Close(websocket.StatusNormalClosure, "")
		}
	}()

	// Require an SDK server to handle sessions. If not provided, return an
	// internal error — callers (main/test) should construct the MCP server and
	// pass it via NewWebSocketServer.
	if s.sdkServer == nil {
		log.Printf("No SDK server available for WebSocket handling")
		_ = conn.Close(websocket.StatusInternalError, "no sdk server available")
		closed = true
		return
	}

	// Use a short timeout for the initial handshake/connection.
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	transport := &websocketServerTransport{conn: conn, sid: ""}
	if _, err := s.sdkServer.Connect(ctx, transport, nil); err != nil {
		log.Printf("SDK server connect failed: %v", err)
		_ = conn.Close(websocket.StatusInternalError, "internal server error")
		closed = true
		return
	}
	// SDK manages the session and the connection lifecycle now.
	closed = true
}
