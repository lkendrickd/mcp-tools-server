package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"mcp-tools-server/pkg/tools"
)

// MCPServer handles MCP protocol communication
type MCPServer struct {
	Tools  map[string]tools.Tool
	logger *slog.Logger
}

// NewMCPServer creates a new MCP server
func NewMCPServer(registry *tools.ToolRegistry, logger *slog.Logger) *MCPServer {
	server := &MCPServer{
		Tools:  make(map[string]tools.Tool),
		logger: logger,
	}

	// Create all available tools from the registry
	availableTools, err := registry.CreateAllAvailable(logger)
	if err != nil {
		logger.Error("Failed to create tools from registry", "error", err)
		// Continue with empty tools map - server will still work but with no tools
		return server
	}

	// Register tools by their actual names
	for _, tool := range availableTools {
		server.Tools[tool.Name()] = tool
	}

	logger.Info("Registered tools", "count", len(server.Tools))
	return server
}

// Start begins the MCP server, reading from stdin and writing to stdout
func (s *MCPServer) Start(ctx context.Context) error {
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
	initResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": s.getAvailableTools(),
			},
			"serverInfo": map[string]interface{}{
				"name":    "mcp-tools-server",
				"version": "1.0.0",
			},
		},
	}

	if err := s.sendResponse(initResponse); err != nil {
		s.logger.Error("Failed to send initialize response", "error", err)
		return fmt.Errorf("failed to send initialize response: %w", err)
	}

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

// getAvailableTools returns the list of available tools
func (s *MCPServer) getAvailableTools() []map[string]interface{} {
	var tools []map[string]interface{}
	for _, tool := range s.Tools {
		schema := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}

		tools = append(tools, map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"inputSchema": schema,
		})
	}
	return tools
}

// handleMessage processes incoming MCP messages
func (s *MCPServer) handleMessage(message map[string]interface{}) error {
	method, ok := message["method"].(string)
	if !ok {
		s.logger.Error("Failed to handle message", "error", fmt.Errorf("invalid message: missing method"))
		return fmt.Errorf("invalid message: missing method")
	}

	id, hasId := message["id"]

	switch method {
	case "initialized":
		// Notification, no response needed
		s.logger.Info("Client initialized")
		return nil
	case "tools/list":
		return s.handleToolsList(id)
	case "tools/call":
		return s.handleToolsCall(message, id)
	default:
		if hasId {
			return s.sendError(id, -32601, "Method not found")
		}
		// Unknown notification, ignore
		s.logger.Warn("Ignoring unknown notification", "method", method)
		return nil
	}
}

// handleToolsList responds to tools/list requests
func (s *MCPServer) handleToolsList(id interface{}) error {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"tools": s.getAvailableTools(),
		},
	}
	return s.sendResponse(response)
}

// handleToolsCall handles tool execution requests
func (s *MCPServer) handleToolsCall(message map[string]interface{}, id interface{}) error {
	params, ok := message["params"].(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid params in tools/call", "error", fmt.Errorf("invalid params"))
		return s.sendError(id, -32602, "Invalid params")
	}

	name, ok := params["name"].(string)
	if !ok {
		s.logger.Error("Missing tool name in tools/call", "error", fmt.Errorf("missing tool name"))
		return s.sendError(id, -32602, "Missing tool name")
	}

	tool, exists := s.Tools[name]
	if !exists {
		return s.sendError(id, -32601, "Tool not found")
	}

	// Extract arguments if present
	arguments, _ := params["arguments"].(map[string]interface{})

	result, err := tool.Execute(arguments)
	if err != nil {
		return s.sendError(id, -32000, err.Error())
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	return s.sendResponse(response)
}

// sendResponse sends a JSON-RPC response
func (s *MCPServer) sendResponse(response map[string]interface{}) error {
	return json.NewEncoder(os.Stdout).Encode(response)
}

// sendError sends a JSON-RPC error response
func (s *MCPServer) sendError(id interface{}, code int, message string) error {
	s.logger.Error("Sending error response", "id", id, "code", code, "message", message)
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	return s.sendResponse(response)
}
