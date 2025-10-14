package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"mcp-tools-server/internal/config"
)

// WebSocketServer handles WebSocket connections.
type WebSocketServer struct {
	config     *config.ServerConfig
	processor  *JSONRPCProcessor
	httpServer *http.Server
}

// NewWebSocketServer creates a new WebSocket server.
func NewWebSocketServer(cfg *config.ServerConfig, processor *JSONRPCProcessor) *WebSocketServer {
	return &WebSocketServer{
		config:    cfg,
		processor: processor,
	}
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
	defer conn.Close(websocket.StatusInternalError, "internal server error")

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	for {
		var request map[string]interface{}
		err := wsjson.Read(ctx, conn, &request)
		if err != nil {
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				if closeErr.Code == websocket.StatusNormalClosure {
					return
				}
			}
			log.Printf("Failed to read from WebSocket: %v", err)
			return
		}

		response := s.processor.Process(r.Context(), request)

		err = wsjson.Write(ctx, conn, response)
		if err != nil {
			log.Printf("Failed to write to WebSocket: %v", err)
			return
		}
	}
}
