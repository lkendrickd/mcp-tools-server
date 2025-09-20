package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// MCPServer handles MCP protocol communication over stdio.
type MCPServer struct {
	logger    *slog.Logger
	processor *JSONRPCProcessor
}

// NewMCPServer creates a new MCP server.
func NewMCPServer(toolService *ToolService, logger *slog.Logger) *MCPServer {
	return &MCPServer{
		logger:    logger,
		processor: NewJSONRPCProcessor(toolService, logger),
	}
}

// Start begins the MCP server, reading from stdin and writing to stdout
func (s *MCPServer) Start(ctx context.Context) error {
	s.logger.Info("Starting MCP server")
	decoder := json.NewDecoder(os.Stdin)

	// Wait for initialize request first
	var initMessage map[string]interface{}
	if err := decoder.Decode(&initMessage); err != nil {
		return fmt.Errorf("failed to read initialize request: %w", err)
	}

	// Validate it's an initialize request
	method, ok := initMessage["method"].(string)
	if !ok || method != "initialize" {
		return fmt.Errorf("expected initialize request, got: %v", initMessage["method"])
	}

	id := initMessage["id"]
	initResponse := s.processor.HandleInitialize(id)

	if err := s.sendResponse(initResponse); err != nil {
		s.logger.Error("Failed to send initialize response", "error", err)
		return fmt.Errorf("failed to send initialize response: %w", err)
	}

	s.logger.Info("MCP server is up and ready for requests")

	// Main message loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var message map[string]interface{}
			if err := decoder.Decode(&message); err != nil {
				s.logger.Error("Failed to decode message", "error", err)
				return fmt.Errorf("failed to decode message: %w", err)
			}

			if err := s.handleMessage(message); err != nil {
				s.logger.Error("Failed to handle message", "error", err)
				return fmt.Errorf("failed to handle message: %w", err)
			}
		}
	}
}

// handleMessage processes incoming MCP messages
func (s *MCPServer) handleMessage(message map[string]interface{}) error {
	method, ok := message["method"].(string)
	if !ok {
		errText := "invalid message: missing method"
		s.logger.Error("Failed to handle message", "error", errText)
		// Try to get ID to send a proper error response
		id, hasId := message["id"]
		if hasId {
			return s.sendResponse(s.processor.CreateErrorResponse(id, -32600, "Invalid Request"))
		}
		return fmt.Errorf("%s", errText)
	}

	id, hasId := message["id"]
	var response *JSONRPCResponse

	switch method {
	case "initialized":
		// Notification, no response needed
		s.logger.Info("Client initialized")
		return nil
	case "tools/list":
		response = s.processor.HandleToolsList(id)
	case "tools/call":
		params, ok := message["params"].(map[string]interface{})
		if !ok {
			response = s.processor.CreateErrorResponse(id, -32602, "Invalid params")
		} else {
			response = s.processor.HandleToolsCall(params, id)
		}
	default:
		if hasId {
			response = s.processor.CreateErrorResponse(id, -32601, "Method not found")
		} else {
			// Unknown notification, ignore
			s.logger.Warn("Ignoring unknown notification", "method", method)
			return nil
		}
	}

	return s.sendResponse(response)
}

// sendResponse sends a JSON-RPC response
func (s *MCPServer) sendResponse(response interface{}) error {
	return json.NewEncoder(os.Stdout).Encode(response)
}
