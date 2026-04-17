# Development Dockerfile with hot reload using Air
# Includes AI CLI tools: Claude Code, Codex, Gemini CLI, OpenCode, Loopal
# All AI CLI tools are pre-configured for headless/non-interactive mode
#
# Debian trixie because Loopal's release binary requires glibc 2.39+.
FROM golang:1.25-trixie

WORKDIR /app

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Install base packages and build tools for native modules
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    tzdata \
    bash \
    curl \
    wget \
    openssh-client \
    openssl \
    python3 \
    # Build tools for native node modules (node-gyp)
    build-essential \
    g++ \
    make \
    # sudo for runner user
    sudo \
    # JSON processing in scripts
    jq \
    # file for binary detection
    file \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 20 (LTS) from NodeSource (Gemini CLI requires Node 20+)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

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

# 5. Loopal - Self-built AI coding agent
#    Install script has a trap/scope bug (unbound $tmpdir in EXIT handler);
#    download, patch, and run.
RUN curl -fsSL -o /tmp/loopal-install.sh \
      https://raw.githubusercontent.com/AgentsMesh/Loopal/main/install/install.sh && \
    sed -i "s/trap 'rm -rf \"\$tmpdir\"' EXIT/trap 'rm -rf \"\${tmpdir:-}\"' EXIT/" /tmp/loopal-install.sh && \
    INSTALL_DIR=/usr/local/bin bash /tmp/loopal-install.sh && \
    rm -f /tmp/loopal-install.sh

# Verify installations
RUN echo "=== Verifying AI CLI installations ===" && \
    claude --version && \
    codex --version && \
    gemini --version && \
    which opencode && echo "OpenCode installed at $(which opencode)" && \
    loopal --version && \
    echo "=== All AI CLI tools installed ==="

# ============================================
# Create non-root user for security
# ============================================

# Create runner user with home directory
RUN groupadd -g 1000 runner && \
    useradd -u 1000 -g runner -m -d /home/runner -s /bin/bash runner && \
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
    mkdir -p /home/runner/.claude && chown -R runner:runner /home/runner/.claude && \
    mkdir -p /home/runner/.codex && chown -R runner:runner /home/runner/.codex && \
    mkdir -p /home/runner/.gemini && chown -R runner:runner /home/runner/.gemini && \
    mkdir -p /home/runner/.opencode && chown -R runner:runner /home/runner/.opencode && \
    mkdir -p /home/runner/.loopal && chown -R runner:runner /home/runner/.loopal

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

# Copy agentfile module (required by go.mod replace directive)
WORKDIR /agentfile
COPY --chown=runner:runner agentfile/go.mod agentfile/go.sum ./
RUN chown -R runner:runner /agentfile

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
