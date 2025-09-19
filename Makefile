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

.PHONY: all run run-http test clean lint wire version help

all: build

# Build the application if source files have changed
build: $(BINARY_PATH)

$(BINARY_PATH): $(GO_FILES)
	@echo "Building $(BINARY_NAME) in $(BUILD_DIR)..."
	@echo "Version: $(VERSION), Build Time: $(BUILD_TIME), Git: $(GIT_COMMIT)"
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) $(GO_BUILD_FLAGS) -o $(BINARY_PATH) ./cmd/server

# Run the combined MCP and HTTP server
run: build
	@echo "Starting server from $(BUILD_DIR)..."
	$(BINARY_PATH)

# Run the HTTP-only server for testing
run-http: build
	@echo "Starting HTTP-only server from $(BUILD_DIR)..."
	$(BINARY_PATH) http-only

# Run all tests
test:
	@echo "Running tests..."
	$(GO_TEST) ./...

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	$(GO_CLEAN)
	rm -f $(BINARY_PATH)
	rm -f *.out *.html

# Run the linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Generate wire_gen.go
wire:
	@echo "Generating wire dependency injection file..."
	$(GO_GENERATE) ./...

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
	@echo "  build        Build the application binary with LDFLAGS"
	@echo "  run          Run the combined MCP and HTTP server"
	@echo "  run-http     Run the HTTP-only server"
	@echo "  test         Run all tests"
	@echo "  clean        Remove binary, coverage files (.out, .html)"
	@echo "  lint         Run the Go linter"
	@echo "  wire         Generate dependency injection files"
	@echo "  docker-build Build Docker image with LDFLAGS"
	@echo "  docker-run   Run Docker container"
	@echo "  docker-clean Clean Docker resources"
	@echo "  docker-push  Push Docker image to registry"
	@echo "  version      Show version and build information"
	@echo "  help         Show this help message"
	@echo ""
	@echo "Binary flags:"
	@echo "  --version    Show version information from the binary"
