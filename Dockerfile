# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (needed for go modules)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with LDFLAGS
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X mcp-tools-server/internal/version.ldflagsVersion=${VERSION} -X mcp-tools-server/internal/version.ldflagsBuildTime=${BUILD_TIME} -X mcp-tools-server/internal/version.ldflagsGitCommit=${GIT_COMMIT} -s -w" \
    -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user and dedicated app directory
RUN addgroup -g 1001 appgroup && \
    adduser -D -u 1001 -G appgroup mcpuser && \
    mkdir -p /app && \
    chown mcpuser:appgroup /app

# Set working directory
WORKDIR /app

# Copy the binary and version file from builder stage
COPY --from=builder --chown=mcpuser:appgroup /app/server .
COPY --from=builder --chown=mcpuser:appgroup /app/version .

# Switch to non-root user
USER mcpuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application in combined MCP and HTTP server mode
ENTRYPOINT ["./server"]
CMD ["http-only"]