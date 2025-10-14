# Makefile for the Go MCP Tools Server

# Go parameters
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_RUN=$(GO_CMD) run
GO_TEST=$(GO_CMD) test
GO_CLEAN=$(GO_CMD) clean
GO_GENERATE=$(GO_CMD) generate

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Linker flags for optimization and metadata
LDFLAGS=\
	-X mcp-tools-server/internal/version.ldflagsVersion=$(VERSION) \
	-X mcp-tools-server/internal/version.ldflagsBuildTime=$(BUILD_TIME) \
	-X mcp-tools-server/internal/version.ldflagsGitCommit=$(GIT_COMMIT) \
	-s -w

# Build flags
GO_BUILD_FLAGS=-ldflags "$(LDFLAGS)"

# Binary name and path
BINARY_NAME=server
BUILD_DIR=build
BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)

# Docker settings
DOCKER_IMAGE_NAME=mcp-tools-server
DOCKER_TAG ?= $(VERSION)

# Source files
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: all run run-http run-mcp run-streamable test-streamable test-stream test clean lint wire version help coverage

all: help

# Build the application if source files have changed
build: $(BINARY_PATH)

$(BINARY_PATH): $(GO_FILES)
	@echo "Building $(BINARY_NAME) in $(BUILD_DIR)..."
	@echo "Version: $(VERSION), Build Time: $(BUILD_TIME), Git: $(GIT_COMMIT)"
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) $(GO_BUILD_FLAGS) -o $(BINARY_PATH) ./cmd/server


# Run the combined MCP and HTTP server (default)
run: build
	@echo "Starting all servers from $(BUILD_DIR)..."
	$(BINARY_PATH) --all

# Run all servers explicitly
run-all: build
	@echo "Starting all servers from $(BUILD_DIR)..."
	$(BINARY_PATH) --all

# Run the HTTP-only server for testing
run-http: build
	@echo "Starting HTTP-only server from $(BUILD_DIR)..."
	$(BINARY_PATH) --http

# Run only the MCP server for local usage
run-mcp: build
	@echo "Starting MCP-only server from $(BUILD_DIR)..."
	$(BINARY_PATH) --mcp

# Run only the Streamable HTTP MCP server
run-streamable: build
	@echo "Starting Streamable HTTP MCP server from $(BUILD_DIR)..."
	$(BINARY_PATH) --streamable

# Run only the WebSocket server
run-websocket: build
	@echo "Starting WebSocket server from $(BUILD_DIR)..."
	$(BINARY_PATH) --websocket

# Test the streamable HTTP MCP server
test-streamable: build
	@echo "Testing Streamable HTTP MCP server..."
	$(GO_RUN) test_stream_client.go

# Test the streamable HTTP MCP server with shell script
test-stream: build
	@echo "Testing Streamable HTTP MCP server with shell script..."
	./test_stream.sh

# Run all tests
test:
	@echo "Running tests..."
	$(GO_TEST) ./...

# Run tests with coverage and create a coverage report
coverage:
	@echo "Running tests with coverage..."
	$(GO_TEST) -coverprofile=coverage.out ./...
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -n 1

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	$(GO_CLEAN)
	rm -f $(BINARY_PATH)
	rm -f *.out *.html

# Run the linter
lint:
	@echo "Running linter..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin; \
	fi
	@$(shell go env GOPATH)/bin/golangci-lint run


# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE_NAME):latest \
		.

# Run Docker container
docker-run:
	@echo "Running Docker container $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)..."
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

# Clean Docker resources
docker-clean:
	@echo "Cleaning Docker resources..."
	docker rmi $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) $(DOCKER_IMAGE_NAME):latest 2>/dev/null || true
	docker system prune -f

# Push Docker image
docker-push:
	@echo "Pushing Docker image $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)..."
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE_NAME):latest

# Show version information
version: build
	@echo "Version information:"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo ""
	@echo "Binary information:"
	@ls -lh $(BINARY_PATH)
	@echo ""
	@echo "To show version from binary, add a version command to your app"

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build          Build the application binary with LDFLAGS"
	@echo "  run            Run the combined MCP and HTTP server"
	@echo "  run-http       Run the HTTP-only server"
	@echo "  run-mcp        Run the MCP-only server"
	@echo "  run-streamable Run the Streamable HTTP MCP server"
	@echo "  run-websocket  Run the WebSocket-only server"
	@echo "  test-streamable Test the Streamable HTTP MCP server (Go client)"
	@echo "  test-stream      Test the Streamable HTTP MCP server (shell script)"
	@echo "  test           Run all tests"
	@echo "  clean          Remove binary, coverage files (.out, .html)"
	@echo "  lint           Run the Go linter"
	@echo "  wire           Generate dependency injection files"
	@echo "  docker-build   Build Docker image with LDFLAGS"
	@echo "  docker-run     Run Docker container"
	@echo "  docker-clean   Clean Docker resources"
	@echo "  docker-push    Push Docker image to registry"
	@echo "  version        Show version and build information"
	@echo "  help           Show this help message"
	@echo ""
	@echo "Binary flags:"
	@echo "  --version    Show version information from the binary"
