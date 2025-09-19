# Developer Guide

## Architecture Overview

This document provides a detailed explanation of the MCP Tools Server architecture, code flow, and how components interact when a tool is needed.

## Application Startup Flow

The journey begins in `cmd/server/main.go`:

```go
// Create core components
cfg := config.NewServerConfig()
registry := tools.NewToolRegistry()
toolService, err := server.NewToolService(registry, logger)
```

**Key Components Created:**
- **ServerConfig**: Reads HTTP_PORT (default 8080) and SHUTDOWN_TIMEOUT (default 30s) from environment
- **ToolRegistry**: Manages tool creation and registration
- **ToolService**: Provides high-level tool execution interface

## Tool Registration and Discovery

The `ToolRegistry` in `pkg/tools/tool.go` automatically registers all available tools:

```go
func (tr *ToolRegistry) registerBuiltinTools() {
    tr.Register("uuid_gen", func(logger *slog.Logger, config map[string]string) (Tool, error) {
        return NewUUIDGen(logger), nil
    })
}
```

**Tool Interface** (`pkg/tools/tool.go`):
```go
type Tool interface {
    Name() string
    Description() string
    Execute(args map[string]interface{}) (map[string]interface{}, error)
}
```

## Tool Service Layer

The `ToolService` in `internal/server/tool_service.go` acts as the bridge between servers and tools:

```go
func NewToolService(registry *tools.ToolRegistry, logger *slog.Logger) (*ToolService, error) {
    // Creates all available tools from registry
    availableTools, err := registry.CreateAllAvailable(logger)

    // Stores tools in a map for quick lookup
    for _, tool := range availableTools {
        service.tools[tool.Name()] = tool
    }
}
```

**Key Methods:**
- `ListTools()`: Returns tool names and descriptions
- `ExecuteTool(name, args)`: Executes a specific tool
- `GetTools()`: Returns all registered tools

## Server Implementations

### MCP Server (`internal/server/mcp_server.go`)
- **Protocol**: JSON-RPC 2.0 over stdin/stdout
- **Initialization**: Responds to `initialize` request with tool capabilities
- **Tool Execution**: Handles `tools/call` requests

**MCP Flow:**
1. Client sends `initialize` → Server responds with available tools
2. Client sends `tools/call` → Server executes via `toolService.ExecuteTool()`
3. Server returns JSON-RPC response with tool results

### HTTP Server (`internal/server/http_server.go`)
- **Endpoints**:
  - `GET /api/uuid` - Execute UUID generation
  - `GET /api/list` - List available tools
  - `GET /health` - Health check
  - `GET /` - Server info

**HTTP Flow:**
1. Client makes HTTP request to endpoint
2. Handler calls `toolService.ExecuteTool()`
3. Results encoded as JSON response

## Tool Execution Example

Taking the UUID generator (`pkg/tools/uuid_gen.go`) as an example:

```go
func (g *UUIDGen) Execute(args map[string]interface{}) (map[string]interface{}, error) {
    uuid, err := g.GenerateUUID()
    if err != nil {
        return map[string]interface{}{"error": err.Error()}, err
    }
    return map[string]interface{}{"uuid": uuid}, nil
}
```

**Complete Flow for a Tool Request:**

### MCP Path:
```
Client → JSON-RPC "tools/call" → MCPServer.handleToolsCall()
                                     ↓
            toolService.ExecuteTool("generate_uuid", args)
                                     ↓
   UUIDGen.Execute() → GenerateUUID() → Return result
                                     ↓
              JSON-RPC Response → Client
```

### HTTP Path:
```
Client → GET /api/uuid → HTTPServer.handleUUID()
                             ↓
         toolService.ExecuteTool("generate_uuid", nil)
                             ↓
   UUIDGen.Execute() → GenerateUUID() → Return result
                             ↓
                JSON Response → Client
```

## Server Modes

The application supports three modes:
- **HTTP Only**: `go run ./cmd/server --http`
- **MCP Only**: `go run ./cmd/server --mcp`
- **Both**: `go run ./cmd/server` (default)

## Key Design Patterns

- **Dependency Injection**: Tools receive logger and config during creation
- **Interface Segregation**: Clean `Tool` interface for extensibility
- **Registry Pattern**: `ToolRegistry` for tool discovery and creation
- **Service Layer**: `ToolService` abstracts tool execution from server protocols
- **Graceful Shutdown**: Combined server handles SIGINT/SIGTERM with timeouts

## Adding New Tools

To add a new tool to the MCP Tools Server:

### 1. Create Tool Implementation

Create a new file in `pkg/tools/` that implements the `Tool` interface:

```go
package tools

import (
    "log/slog"
)

type MyTool struct {
    logger *slog.Logger
}

func NewMyTool(logger *slog.Logger) *MyTool {
    return &MyTool{
        logger: logger,
    }
}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "Description of what my tool does"
}

func (t *MyTool) Execute(args map[string]interface{}) (map[string]interface{}, error) {
    // Implement your tool logic here
    result := map[string]interface{}{
        "output": "tool result",
    }
    return result, nil
}
```

### 2. Register the Tool

Add the tool to the registry in `pkg/tools/tool.go`:

```go
func (tr *ToolRegistry) registerBuiltinTools() {
    // ... existing tools ...

    tr.Register("my_tool", func(logger *slog.Logger, config map[string]string) (Tool, error) {
        return NewMyTool(logger), nil
    })
}
```

### 3. Add HTTP Endpoint (Optional)

If you want HTTP access, add an endpoint in `internal/server/http_server.go`:

```go
func NewHTTPServer(toolService *ToolService, port int, logger *slog.Logger) *HTTPServer {
    // ... existing setup ...

    // Add your new endpoint
    mux.HandleFunc("/api/my-tool", httpServer.handleMyTool)

    return httpServer
}

func (s *HTTPServer) handleMyTool(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    result, err := s.toolService.ExecuteTool("my_tool", nil)
    if err != nil {
        http.Error(w, "Tool execution failed", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### 4. Test Your Tool

```bash
# Test via HTTP
curl http://localhost:8080/api/my-tool

# Test via MCP (requires MCP client)
# The tool will automatically appear in tools/list
```

## Configuration

Configuration is managed through environment variables:

- `HTTP_PORT`: HTTP server port (default: 8080)
- `SHUTDOWN_TIMEOUT`: Graceful shutdown timeout in seconds (default: 30)

## Testing Strategy

The project includes comprehensive testing:

- **Unit Tests**: Individual component testing
- **Integration Tests**: Server and tool interaction testing
- **Contract Tests**: Protocol compliance testing

Run tests with:
```bash
make test
```

## Build and Deployment

### Local Development
```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
```

### Docker
```bash
make docker-build  # Build Docker image
make docker-run    # Run in container
```

### Production Deployment
The server can be deployed as:
- Standalone binary
- Docker container
- System service

## Error Handling

The server implements comprehensive error handling:

- **Tool Execution Errors**: Propagated through service layer
- **Network Errors**: HTTP server returns appropriate status codes
- **Protocol Errors**: MCP server returns JSON-RPC error responses
- **Configuration Errors**: Logged and handled gracefully

## Logging

Structured logging is implemented using `slog`:

- **Info Level**: Normal operations, tool executions
- **Warn Level**: Non-critical issues, method not allowed
- **Error Level**: Failures, encoding errors, tool execution failures

## Performance Considerations

- **Concurrent Requests**: Both servers handle multiple simultaneous requests
- **Memory Efficient**: Tools are created once at startup
- **Fast Tool Lookup**: Hash map provides O(1) tool access
- **Minimal Dependencies**: Small binary size and fast startup

## Security

- **Input Validation**: HTTP endpoints validate request methods
- **Error Information**: Sensitive details not exposed in responses
- **Resource Limits**: No explicit limits (consider adding for production)
- **Environment Variables**: Configuration through secure env vars

## Monitoring and Observability

- **Health Endpoint**: `/health` for load balancer checks
- **Version Endpoint**: `/` includes version and build info
- **Structured Logs**: JSON-formatted logs for log aggregation
- **Metrics**: Basic request counting (can be extended)

## Future Enhancements

Potential areas for improvement:

- **Authentication/Authorization**: Add API key or OAuth support
- **Rate Limiting**: Prevent abuse of tool endpoints
- **Metrics Collection**: Add Prometheus metrics
- **Configuration File**: Support YAML/JSON config files
- **Tool Dependencies**: Support tools with external dependencies
- **Caching**: Add result caching for expensive operations
- **WebSocket Support**: Real-time tool execution updates</content>
<parameter name="filePath">/home/dennis/go/src/github.com/lkendrickd/mcp-tools-server/DEVELOPER_GUIDE.md
