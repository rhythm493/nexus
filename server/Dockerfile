# Multi-stage build for Nexus Go server

# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o nexus ./cmd/server

# Stage 2: Runtime image
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    liquidsoap \
    ffmpeg \
    curl \
    python3 \
    python3-pip \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Install yt-dlp and uv in a virtual environment (avoids --break-system-packages)
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install --no-cache-dir yt-dlp uv

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /build/nexus /app/

# Copy scripts and MCP server configs (config.yaml should come from volume mount or env vars)
COPY scripts/ /app/scripts/
COPY mcp-servers/ /app/mcp-servers/

# Create necessary directories with explicit permissions
RUN mkdir -p /app/config /app/data/library /app/data/bin /app/certs /app/streams && \
    chmod 755 /app/config /app/data /app/data/library /app/data/bin /app/certs /app/streams

# Set environment variables
ENV LOG_LEVEL=info \
    PORT=8443 \
    RADIO_ENABLED=true \
    YOUTUBE_ENABLED=true

# Expose ports
EXPOSE 8443 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -k -f https://localhost:8443/api/v1/health || exit 1

# Run the application
CMD ["/app/nexus"]
