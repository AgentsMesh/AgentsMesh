# Development Dockerfile with hot reload using Air
# Includes AI CLI tools: Claude Code, Codex, Gemini CLI, OpenCode
# All AI CLI tools are pre-configured for headless/non-interactive mode
FROM docker.1ms.run/library/golang:1.25-alpine

WORKDIR /app

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Install base packages and build tools for native modules
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    bash \
    curl \
    openssh-client \
    openssl \
    python3 \
    nodejs \
    npm \
    # Build tools for native node modules (node-gyp)
    build-base \
    g++ \
    make \
    linux-headers \
    # For non-root user
    shadow \
    sudo \
    # For JSON processing in scripts
    jq

# ============================================
# Install AI CLI Tools (as root, before user switch)
# ============================================

# 1. Claude Code - Anthropic's AI coding assistant
RUN npm install -g @anthropic-ai/claude-code

# 2. OpenAI Codex CLI - OpenAI's coding agent
RUN npm install -g @openai/codex

# 3. Gemini CLI - Google's AI coding assistant
RUN npm install -g @google/gemini-cli

# 4. OpenCode - Open source AI coding agent
RUN npm install -g opencode-ai

# Verify installations
RUN echo "=== Verifying AI CLI installations ===" && \
    claude --version && \
    codex --version && \
    gemini --version && \
    which opencode && echo "OpenCode installed at $(which opencode)" && \
    echo "=== All AI CLI tools installed ==="

# ============================================
# Create non-root user for security
# ============================================

# Create runner user with home directory
RUN addgroup -g 1000 runner && \
    adduser -u 1000 -G runner -h /home/runner -s /bin/bash -D runner && \
    # Give runner user ownership of app directory
    chown -R runner:runner /app && \
    # Create workspace directories (volume mount point + fallback)
    mkdir -p /workspace && \
    chown -R runner:runner /workspace && \
    mkdir -p /tmp/agentsmesh-workspace && \
    chown -R runner:runner /tmp/agentsmesh-workspace && \
    # Create .agentsmesh config directory (note: with 's')
    mkdir -p /home/runner/.agentsmesh && \
    chown -R runner:runner /home/runner/.agentsmesh && \
    # Create go build cache directory
    mkdir -p /home/runner/.cache/go-build && \
    chown -R runner:runner /home/runner/.cache && \
    # Copy air binary to accessible location (installed via go install to /go/bin)
    cp /go/bin/air /usr/local/bin/air && \
    # ============================================
    # Create AI CLI config directories
    # ============================================
    # Claude Code config directory
    mkdir -p /home/runner/.claude && \
    chown -R runner:runner /home/runner/.claude && \
    # OpenAI Codex config directory
    mkdir -p /home/runner/.codex && \
    chown -R runner:runner /home/runner/.codex && \
    # Gemini CLI config directory
    mkdir -p /home/runner/.gemini && \
    chown -R runner:runner /home/runner/.gemini && \
    # OpenCode config directory
    mkdir -p /home/runner/.opencode && \
    chown -R runner:runner /home/runner/.opencode

# ============================================
# Copy AI CLI pre-configured settings
# These settings enable headless/non-interactive mode
# ============================================
COPY --chown=runner:runner deploy/dev/ai-cli-configs/claude/settings.json /home/runner/.claude/settings.json
COPY --chown=runner:runner deploy/dev/ai-cli-configs/codex/config.toml /home/runner/.codex/config.toml
COPY --chown=runner:runner deploy/dev/ai-cli-configs/gemini/settings.json /home/runner/.gemini/settings.json

# ============================================
# Go module setup
# ============================================

# Copy proto module first (required by go.mod replace directive)
WORKDIR /proto
COPY --chown=runner:runner proto/go.mod proto/go.sum ./
RUN chown -R runner:runner /proto

# Copy runner go mod files
WORKDIR /app
COPY --chown=runner:runner runner/go.mod runner/go.sum ./
RUN chown -R runner:runner /go
USER runner
RUN go mod download

# Source code will be mounted as volume

# Note: Runner connects outbound to Backend via gRPC+mTLS
# No inbound port needed (port 9090 was for legacy WebSocket)
EXPOSE 9090

# Entrypoint script mounted via docker-compose volume
# Default command (can be overridden)
CMD ["air", "-c", ".air.toml"]
