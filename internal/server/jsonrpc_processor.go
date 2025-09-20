package server

import (
	"fmt"
	"log/slog"
)

// JSONRPCProcessor handles the logic for JSON-RPC messages, independent of transport.
type JSONRPCProcessor struct {
	toolService *ToolService
	logger      *slog.Logger
}

// NewJSONRPCProcessor creates a new JSONRPCProcessor.
func NewJSONRPCProcessor(toolService *ToolService, logger *slog.Logger) *JSONRPCProcessor {
	return &JSONRPCProcessor{
		toolService: toolService,
		logger:      logger,
	}
}

// --- Response Structs ---

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]interface{} `json:"serverInfo"`
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorObject `json:"error,omitempty"`
}

type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- Public Methods ---

// HandleInitialize creates the response for an "initialize" request.
func (p *JSONRPCProcessor) HandleInitialize(id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": p.getAvailableTools(),
			},
			ServerInfo: map[string]interface{}{
				"name":    "mcp-tools-server",
				"version": "1.0.0",
			},
		},
	}
}

// HandleToolsList creates the response for a "tools/list" request.
func (p *JSONRPCProcessor) HandleToolsList(id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": p.getAvailableTools(),
		},
	}
}

// HandleToolsCall handles a "tools/call" request and returns a response.
func (p *JSONRPCProcessor) HandleToolsCall(params map[string]interface{}, id interface{}) *JSONRPCResponse {
	name, ok := params["name"].(string)
	if !ok {
		p.logger.Error("Missing tool name in tools/call")
		return p.CreateErrorResponse(id, -32602, "Invalid params: Missing tool name")
	}

	arguments, _ := params["arguments"].(map[string]interface{})

	result, err := p.toolService.ExecuteTool(name, arguments)
	if err != nil {
		p.logger.Error("Error executing tool", "tool", name, "error", err)
		return p.CreateErrorResponse(id, -32000, fmt.Sprintf("Tool execution error: %s", err.Error()))
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// CreateErrorResponse creates a standardized JSON-RPC error response.
func (p *JSONRPCProcessor) CreateErrorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	p.logger.Error("Sending error response", "id", id, "code", code, "message", message)
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorObject{
			Code:    code,
			Message: message,
		},
	}
}

// --- Private Helpers ---

// getAvailableTools returns the list of available tools in the required format.
func (p *JSONRPCProcessor) getAvailableTools() []ToolDefinition {
	var tools []ToolDefinition
	for _, tool := range p.toolService.GetTools() {
		// For now, schema is a generic object. This could be expanded later.
		schema := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}

		tools = append(tools, ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: schema,
		})
	}
	return tools
}
