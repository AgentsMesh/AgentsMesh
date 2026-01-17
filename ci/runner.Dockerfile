# Build stage
ARG REGISTRY=registry.corp.agentsmesh.ai
FROM ${REGISTRY}/library/golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -tags nocgo -ldflags="-w -s" -o /app/runner ./cmd/runner

# Final stage
ARG REGISTRY=registry.corp.agentsmesh.ai
FROM ${REGISTRY}/library/alpine:3.19

WORKDIR /app

# Install required packages
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    git \
    openssh-client \
    bash \
    curl

# Create non-root user with home directory for git operations
RUN addgroup -g 1000 -S runner && \
    adduser -u 1000 -S runner -G runner -h /home/runner

# Create workspace directory
RUN mkdir -p /workspace && chown runner:runner /workspace

# Copy binary from builder
COPY --from=builder /app/runner /app/runner

# Set ownership
RUN chown -R runner:runner /app

# Switch to non-root user
USER runner

# Set workspace as working directory
WORKDIR /workspace

# Note: Runner connects outbound to Backend via gRPC+mTLS
# No inbound port needed (port 9090 was for legacy WebSocket)
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:9090/health || exit 1

# Run the binary
ENTRYPOINT ["/app/runner"]
