# Build stage
# Build context should be project root (not runner/)
# REGISTRY_PREFIX: Use internal mirror for GitLab CI (e.g., registry.corp.agentsmesh.ai/library/)
#                  Leave empty for Docker Hub (GitHub Actions)
ARG REGISTRY_PREFIX=
ARG GO_VERSION=1.25
FROM ${REGISTRY_PREFIX}golang:${GO_VERSION} AS builder

WORKDIR /app

# Go module proxy (override via --build-arg for faster downloads in specific regions)
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

# Copy proto module first (required by replace directive in go.mod)
COPY proto /proto

# Copy agentfile module (required by replace directive in go.mod)
COPY agentfile/go.mod agentfile/go.sum /agentfile/

# Copy go mod files
COPY runner/go.mod runner/go.sum ./
RUN go mod download

# Copy source code
COPY agentfile/ /agentfile/
COPY runner/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -tags nocgo -ldflags="-w -s" -o /app/runner ./cmd/runner

# Final stage
ARG REGISTRY_PREFIX=
FROM ${REGISTRY_PREFIX}alpine:3.19

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
