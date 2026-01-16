# Development Dockerfile with hot reload using Air
# Includes AI CLI tools: Claude Code, Codex, Gemini CLI, OpenCode
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
    sudo

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
    # Create workspace directory
    mkdir -p /tmp/agentsmesh-workspace && \
    chown -R runner:runner /tmp/agentsmesh-workspace && \
    # Create .agentsmesh config directory (note: with 's')
    mkdir -p /home/runner/.agentsmesh && \
    chown -R runner:runner /home/runner/.agentsmesh && \
    # Create go build cache directory
    mkdir -p /home/runner/.cache/go-build && \
    chown -R runner:runner /home/runner/.cache && \
    # Copy air binary to accessible location (installed via go install to /go/bin)
    cp /go/bin/air /usr/local/bin/air

# ============================================
# Go module setup
# ============================================

# Copy go mod files first for better caching
COPY --chown=runner:runner go.mod go.sum ./
RUN chown -R runner:runner /go
USER runner
RUN go mod download

# Source code will be mounted as volume

# Expose port for WebSocket connections
EXPOSE 9090

# Entrypoint script mounted via docker-compose volume
# Default command (can be overridden)
CMD ["air", "-c", ".air.toml"]
