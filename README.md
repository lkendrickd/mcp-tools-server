<img src="images/logo.png" alt="MCP Tools Server Logo" width="50%">

# MCP Tools Server

A Go-based Model Context Protocol (MCP) server that provides simple tools for AI assistants.

## Features

- **UUID Generation Tool**: Used as an Example. Generates random UUID v4 strings via MCP protocol
- **Multiple Protocol Support**: Works with MCP (stdio), HTTP REST API, Streamable HTTP, and WebSockets.
- **Graceful Shutdown**: Handles system signals properly for clean termination
- **Concurrent Requests**: Supports multiple simultaneous tool calls
- **Comprehensive Testing**: Unit, integration, and contract tests included
- **Makefile Automation**: Convenient build, test, and run commands
- **Extensible Architecture**: Easily add new tools by implementing the Tool interface
- **Tiny Footprint**: A singe 5.8MB compiled binary or a 15MB Docker image
- **Prometheus Metrics**: Built-in metrics for monitoring server performance

## Installation

### Prerequisites

- Go 1.24.6+ (as specified in `go.mod`)
- Make (for using the Makefile commands)

### Setup

```bash
git clone <repository-url>
cd mcp-tools-server
go mod tidy
make build
```

## Usage

### Using Makefile Commands

The project includes a `Makefile` for common operations:

- **`make build`**: Build the application binary.
- **`make run`** or **`make run-all`**: Run all servers (default).
- **`make run-http`**: Run only the HTTP REST server.
- **`make run-mcp`**: Run only the Stdio MCP server.
- **`make run-streamable`**: Run only the Streamable HTTP server.
- **`make run-websocket`**: Run only the WebSocket server.
- **`make test`**: Run all tests.
- **`make clean`**: Remove build artifacts.
- **`make lint`**: Run the Go linter.
- **`make help`**: Show all available commands.

### Running the Server

You can control which servers to run using command-line flags. By default, all four servers (Stdio MCP, HTTP REST, Streamable HTTP, and WebSocket) are enabled. You can also use the `make` targets (`make run-http`, `make run-mcp`, etc.) as a convenient shortcut for these commands.

- **Run all servers (default):**
  ```bash
  ./build/server
  # or
  ./build/server --all
  ```
- **Run only the HTTP REST server:**
  ```bash
  ./build/server --http
  ```
- **Run only the Stdio MCP server:**
  ```bash
  ./build/server --mcp
  ```
- **Run only the Streamable HTTP MCP server:**
  ```bash
  ./build/server --streamable
  ```
  The streamable server runs on port 8081 by default.

- **Run only the WebSocket server:**
  ```bash
  ./build/server --websocket
  ```
  The WebSocket server runs on port 8082 by default.

- **Show version info:**
  ```bash
  ./build/server --version
  ```


### As MCP Server

The server communicates via stdio for MCP clients (typically used by AI assistants):

```bash
./build/server --mcp
```

### Streamable HTTP MCP

The server now supports the official **Streamable HTTP** transport from the MCP specification. This runs on port 8081 by default and provides a single `/mcp` endpoint for all communication.

- **Making a tool call (POST):**
  ```bash
  curl -X POST http://localhost:8081/mcp \
    -H "Content-Type: application/json" \
    -d '{
      "jsonrpc": "2.0",
      "id": 1,
      "method": "tools/call",
      "params": {
        "name": "generate_uuid"
      }
    }'
  ```
  <img src="images/logo.png" alt="MCP Tools Server Logo" width="50%">

  # MCP Tools Server

  A Go-based Model Context Protocol (MCP) server that exposes a small collection of tools over multiple transports. The server uses the official MCP Go SDK (github.com/modelcontextprotocol/go-sdk v1.0.0) to manage sessions, transports, and tool registration.

  ## Key characteristics

  - SDK-backed: The server uses an SDK `mcp.Server` as the central session and tool dispatcher. Transports (stdio, Streamable HTTP, WebSocket) connect into that server.
  - Multiple transports: stdio (MCP), Streamable HTTP (`/mcp`), HTTP REST, and WebSocket (`/ws`).
  - Tool adapter: tools live in `pkg/tools` and are registered with the SDK via `internal/server.ToolService`. Tool outputs are normalized to a consistent JSON object shape.
  - Extensible: add new tools by implementing the `Tool` interface and registering a builder.
  - Observability: basic Prometheus metrics and HTTP health endpoints are provided.

  ## Installation

  ### Prerequisites

  - Go 1.24+
  - GNU Make (optional but used by the included Makefile)

  ### Quick start

  ```bash
  git clone <repository-url>
  cd mcp-tools-server
  go mod download
  make build
  ```

  ## Running the server

  The binary supports flags to enable or disable transports. The `Makefile` contains convenience targets.

  - `make run` or `make run-all` — run all enabled servers
  - `make run-http` — run the REST API only
  - `make run-mcp` — run the stdio (MCP) server only
  - `make run-streamable` — run the Streamable HTTP MCP server only
  - `make run-websocket` — run the WebSocket server only

  By default the binary listens on the following ports (configurable):

  - REST HTTP: 8080
  - Streamable HTTP MCP: 8081 (`/mcp`)
  - WebSocket MCP: 8082 (`/ws`)

  Example — run the Streamable HTTP server and call a tool:

  ```bash
  ./build/server --streamable

  curl -X POST http://localhost:8081/mcp \
    -H "Content-Type: application/json" \
    -d '{
      "jsonrpc": "2.0",
      "id": 1,
      "method": "tools/call",
      "params": {"name": "generate_uuid"}
    }'
  ```

  Example — connect with a WebSocket client:

  ```bash
  # connect to the WebSocket MCP endpoint
  websocat ws://localhost:8082/ws
  ```

  ## HTTP API

  The server exposes a small REST surface used for testing and convenience:

  - `GET /api/uuid` — returns a generated UUID (example tool)
  - `GET /api/list` — lists available tools and descriptions
  - `GET /health` — health check
  - `GET /metrics` — Prometheus metrics

  Example:

  ```bash
  curl http://localhost:8080/api/uuid
  ```

  Response:

  ```json
  {"uuid":"550e8400-e29b-41d4-a716-446655440000"}
  ```

  ## Tools

  Tools are implemented under `pkg/tools`. Each tool implements the `Tool` interface (Name, Description, Execute). The repository includes a `generate_uuid` tool as an example.

  Tool outputs are normalized by the server into a consistent `map[string]interface{}` JSON object shape. This makes consumer parsing simpler and the server resilient to differences in how individual tools return their results.

  ## Development

  Project layout (relevant folders):

  ```
  ├── cmd/server/           # main entry point
  ├── internal/             # server implementation, configuration
  │   ├── config/
  │   └── server/
  ├── pkg/tools/            # tool implementations and builders
  ├── docs/                 # design and developer guides
  ├── build/                # built artifacts
  ```

  ### Tests

  Run unit tests and package checks with:

  ```bash
  make test
  go test ./...
  ```

  ### Linting

  Run the repository linter (this project expects `golangci-lint` to be available):

  ```bash
  /home/dennis/go/bin/golangci-lint run ./...
  ```

  ## Configuration

  Configuration values are defined in `internal/config` and are exposed via environment variables and flags. Common settings include HTTP ports and graceful shutdown timeouts.

  ## Contributing

  Contributions are welcome. Fork, add a feature branch, run tests and linter, and open a pull request.

  ## License

  This project is licensed under the MIT License. See the `LICENSE` file for details.
