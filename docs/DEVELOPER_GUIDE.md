# Developer Guide: Adding New Tools to MCP Tools Server

This guide explains how to add new tools to the MCP Tools Server. We'll use the Brave Web Search tool as a practical example.

Here is the order of files to edit when adding a new tool:

1. **pkg/tools/brave_websearch.go** (new file) - Define the BraveWebSearch struct and implement the Tool interface methods (Name, Description, Execute) to handle API calls and responses.
2. **internal/config/config.go** - Add BraveAPIKey field to ServerConfig and update NewServerConfig to read BRAVE_API_KEY from environment variables for secure API key management.
3. **internal/server/mcp_server.go** - Update NewMCPServer to accept and register the new tool, and modify getAvailableTools to include the proper input schema for the search query parameter.
4. **cmd/server/main.go** - Instantiate the BraveWebSearch tool with the config's API key and pass it to the MCP server for initialization.


## Overview

The MCP Tools Server uses a modular tool system where each tool implements the `tools.Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Execute(arguments map[string]interface{}) (map[string]interface{}, error)
}
```

Tools are registered in the MCP server and automatically become available to MCP clients.

## Step-by-Step Guide to Adding a New Tool

### Step 1: Define the Tool Interface Implementation

Create a new file in `pkg/tools/` for your tool. For the Brave Web Search tool:

```go
package tools

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
)

// BraveWebSearch implements the Tool interface for Brave Search API
type BraveWebSearch struct {
    apiKey string
}

// NewBraveWebSearch creates a new Brave Web Search tool
func NewBraveWebSearch(apiKey string) *BraveWebSearch {
    return &BraveWebSearch{apiKey: apiKey}
}

// Name returns the tool name
func (b *BraveWebSearch) Name() string {
    return "brave_web_search"
}

// Description returns the tool description
func (b *BraveWebSearch) Description() string {
    return "Search the web using Brave Search API"
}

// Execute performs the web search
func (b *BraveWebSearch) Execute(arguments map[string]interface{}) (map[string]interface{}, error) {
    query, ok := arguments["query"].(string)
    if !ok {
        return nil, fmt.Errorf("missing required argument: query")
    }

    // Build the API request
    apiURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s", url.QueryEscape(query))
    
    req, err := http.NewRequest("GET", apiURL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Accept", "application/json")
    req.Header.Set("X-Subscription-Token", b.apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to execute search: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Brave API returned status: %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    return map[string]interface{}{
        "results": result,
    }, nil
}
```

### Step 2: Update Configuration

Add configuration for your tool's dependencies (e.g., API keys) in `internal/config/config.go`:

```go
// ServerConfig holds the configuration for the MCP tools server
type ServerConfig struct {
    // ... existing fields ...
    BraveAPIKey string // Add this field
}

// NewServerConfig creates a new server configuration
func NewServerConfig() *ServerConfig {
    return &ServerConfig{
        // ... existing fields ...
        BraveAPIKey: getEnvString("BRAVE_API_KEY", ""), // Add this
    }
}

// getEnvString reads a string from the environment or returns the default
func getEnvString(key, defaultVal string) string {
    if val, ok := os.LookupEnv(key); ok {
        return val
    }
    return defaultVal
}
```

### Step 3: Register the Tool in the Server

Update `internal/server/mcp_server.go` to register your new tool:

```go
// NewMCPServer creates a new MCP server
func NewMCPServer(uuidGen *tools.UUIDGen, braveSearch *tools.BraveWebSearch, logger *slog.Logger) *MCPServer {
    server := &MCPServer{
        Tools:  make(map[string]tools.Tool),
        logger: logger,
    }
    
    // Register tools
    server.Tools[uuidGen.Name()] = uuidGen
    server.Tools[braveSearch.Name()] = braveSearch // Add this line
    
    return server
}
```

### Step 4: Update Server Initialization

In `cmd/server/main.go`, create and pass the new tool to the server:

```go
func main() {
    // ... existing code - snippet shown for context ...

    cfg := config.NewServerConfig()
    
    // Create tools
    uuidGen := tools.NewUUIDGen()
    braveSearch := tools.NewBraveWebSearch(cfg.BraveAPIKey)
    
    // Create servers
    mcpServer := server.NewMCPServer(uuidGen, braveSearch, logger)
    httpServer := server.NewHTTPServer(uuidGen, cfg.HTTPPort, logger)
    
    // ... rest of main function - refer to main.go ...
}
```

### Step 5: Update Input Schema (Optional)

If your tool has specific input requirements, update the `getAvailableTools()` method in `mcp_server.go` to include proper input schemas:

```go
func (s *MCPServer) getAvailableTools() []map[string]interface{} {
    var tools []map[string]interface{}
    for _, tool := range s.Tools {
        schema := map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{},
        }
        
        // Customize schema based on tool
        if tool.Name() == "brave_web_search" {
            schema["properties"] = map[string]interface{}{
                "query": map[string]interface{}{
                    "type": "string",
                    "description": "The search query",
                },
            }
            schema["required"] = []string{"query"}
        }
        
        tools = append(tools, map[string]interface{}{
            "name":        tool.Name(),
            "description": tool.Description(),
            "inputSchema": schema,
        })
    }
    return tools
}
```

### Step 6: Test Your Tool

1. Set the required environment variables:
   ```bash
   export BRAVE_API_KEY="your-api-key-here"
   ```

2. Build and run the server:
   ```bash
   make build
   ./server
   ```

3. Test with an MCP client or curl:
   ```bash
   curl -X POST http://localhost:8080/tools/call \
     -H "Content-Type: application/json" \
     -d '{
       "name": "brave_web_search",
       "arguments": {"query": "Go programming language"}
     }'
   ```

## Best Practices

1. **Error Handling**: Always return meaningful errors from your tool's `Execute` method.
2. **Input Validation**: Validate required arguments and types.
3. **Configuration**: Use environment variables for sensitive data like API keys.
4. **Logging**: Use the provided logger for debugging and monitoring.
5. **Documentation**: Update this guide and README.md when adding new tools.
6. **Testing**: Add unit tests for your tool in the appropriate test directory.

## Common Patterns

- **API Keys**: Store in environment variables, validate presence.
- **HTTP Clients**: Reuse http.Client instances for efficiency.
- **Timeouts**: Set reasonable timeouts for external API calls.
- **Rate Limiting**: Implement if required by the API.
- **Caching**: Consider caching results for expensive operations.

## Troubleshooting

- **Tool not appearing**: Check that it's registered in `NewMCPServer`.
- **API errors**: Verify API keys and network connectivity.
- **Input validation**: Ensure argument names match the schema.
- **Build errors**: Check import paths and dependencies.

## Next Steps

After implementing your tool:
1. Add comprehensive tests
2. Update documentation
3. Consider adding monitoring/metrics
4. Test with real MCP clients</content>
<parameter name="filePath">/home/dennis/go/src/github.com/lkendrickd/mcp-tools-server/docs/DEVELOPER_GUIDE.md
