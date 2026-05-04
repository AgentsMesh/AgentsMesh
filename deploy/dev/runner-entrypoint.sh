#!/bin/bash
# AgentsMesh Runner — Docker entrypoint (Bazel-built binary mode)
#
# Sets up:
#   1. Wait for backend to come up
#   2. Generate runner mTLS client cert from the dev CA
#   3. Initialize AI CLI configs (Claude Code / Codex / Gemini)
#   4. Write runner config.yaml
#   5. exec the bazel-built runner binary (`agentsmesh-runner`)
#
# This replaces the old `air -c .air.toml` flow — the binary is built
# at image-build time by `bazel build //runner/cmd/runner:runner` and
# COPYed into the image. There's no go toolchain inside the container.

set -e
trap 'echo "✗ entrypoint failed at line $LINENO (exit=$?)" >&2' ERR

BACKEND_URL="${BACKEND_URL:-http://traefik:80}"
GRPC_ENDPOINT="${GRPC_ENDPOINT:-traefik:9443}"
RELAY_BASE_URL="${RELAY_BASE_URL:-ws://traefik:80}"
RUNNER_NODE_ID="${RUNNER_NODE_ID:-dev-runner}"
RUNNER_ORG_SLUG="${RUNNER_ORG_SLUG:-dev-org}"
MAX_CONCURRENT_PODS="${MAX_CONCURRENT_PODS:-10}"
SSL_DIR="${SSL_DIR:-/app/ssl}"

CONFIG_DIR="${HOME}/.agentsmesh"
CERTS_DIR="${CONFIG_DIR}/certs"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"

echo "========================================"
echo "  AgentsMesh Runner Entrypoint (Bazel)"
echo "========================================"
echo "  Backend URL:    $BACKEND_URL"
echo "  gRPC Endpoint:  $GRPC_ENDPOINT"
echo "  Relay Base URL: $RELAY_BASE_URL"
echo "  Node ID:        $RUNNER_NODE_ID"
echo "  Org Slug:       $RUNNER_ORG_SLUG"
echo "  Max Pods:       $MAX_CONCURRENT_PODS"
echo ""

wait_for_backend() {
    echo "等待 Backend 服务就绪..."
    local health_url="${BACKEND_URL}/health"
    # Cold CI runs build the backend Bazel binary on first invocation
    # (~4min for protoc + Go compile). Bumping the bound to 240×2s = 8min
    # so a freshly-cached image doesn't lose the race against runner's
    # restart loop.
    for ((i=1; i<=240; i++)); do
        if wget -q -O /dev/null "$health_url" 2>/dev/null; then
            echo "✓ Backend 服务就绪"
            return 0
        fi
        echo "  等待 Backend... ($i/240)"
        sleep 2
    done
    echo "✗ Backend 服务启动超时" >&2
    exit 1
}

generate_runner_cert() {
    echo "▶ generate_runner_cert: CERTS_DIR=$CERTS_DIR SSL_DIR=$SSL_DIR"
    mkdir -p "$CERTS_DIR"
    ls -la "$CERTS_DIR" >&2 || true
    # All three files must be present and non-empty: a partial state from a
    # previous failed run (e.g. SSL_DIR/ca.crt arrived after `wait_for_backend`
    # returned but before openssl signed the runner cert) leaves zero-byte
    # runner.crt + runner.key while ca.crt is missing entirely. Don't trust
    # that, regenerate from scratch.
    if [[ -s "$CERTS_DIR/runner.crt" && -s "$CERTS_DIR/runner.key" && -s "$CERTS_DIR/ca.crt" ]]; then
        echo "✓ Runner 证书已存在"
        return 0
    fi
    rm -f "$CERTS_DIR/runner.crt" "$CERTS_DIR/runner.key" "$CERTS_DIR/ca.crt" \
          "$CERTS_DIR/ca.srl" "$CERTS_DIR/runner.csr" "$CERTS_DIR/runner_ext.cnf"
    if [[ ! -f "$SSL_DIR/ca.crt" || ! -f "$SSL_DIR/ca.key" ]]; then
        echo "✗ CA 证书未找到: $SSL_DIR" >&2
        ls -la "$SSL_DIR" >&2 || true
        exit 1
    fi
    echo "生成 Runner 客户端证书..."
    openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 \
        -out "$CERTS_DIR/runner.key" 2>/dev/null
    openssl req -new -key "$CERTS_DIR/runner.key" \
        -out "$CERTS_DIR/runner.csr" \
        -subj "/CN=${RUNNER_NODE_ID}/O=${RUNNER_ORG_SLUG}/OU=Runner" 2>/dev/null
    cat > "$CERTS_DIR/runner_ext.cnf" << 'EOF'
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF
    openssl x509 -req -days 365 \
        -in "$CERTS_DIR/runner.csr" \
        -CA "$SSL_DIR/ca.crt" -CAkey "$SSL_DIR/ca.key" \
        -CAserial "$CERTS_DIR/ca.srl" -CAcreateserial \
        -out "$CERTS_DIR/runner.crt" \
        -extfile "$CERTS_DIR/runner_ext.cnf" 2>/dev/null
    cp "$SSL_DIR/ca.crt" "$CERTS_DIR/ca.crt"
    rm -f "$CERTS_DIR/runner.csr" "$CERTS_DIR/runner_ext.cnf"
    chmod 600 "$CERTS_DIR/runner.key"
    chmod 644 "$CERTS_DIR/runner.crt" "$CERTS_DIR/ca.crt"
    echo "✓ Runner 证书生成完成"
}

init_ai_cli_configs() {
    # Claude Code: hasCompletedOnboarding lives in ~/.claude.json (the
    # CLI requires this exact path), so we keep the actual file under
    # ~/.claude/ for volume persistence and symlink ~/.claude.json to it.
    local claude_dir="${HOME}/.claude"
    local claude_actual="${claude_dir}/claude.json"
    local claude_link="${HOME}/.claude.json"
    mkdir -p "$claude_dir"
    if [[ ! -f "$claude_actual" ]]; then
        cat > "$claude_actual" << 'EOF'
{
  "hasCompletedOnboarding": true,
  "theme": "dark",
  "autoUpdaterStatus": "disabled",
  "shiftEnterKeyBindingInstalled": true
}
EOF
    fi
    if [[ ! -L "$claude_link" ]]; then
        rm -f "$claude_link"
        ln -s "$claude_actual" "$claude_link"
    fi
    if [[ ! -f "$claude_dir/settings.json" ]]; then
        cat > "$claude_dir/settings.json" << 'EOF'
{
  "permissions": {
    "allow": ["Bash(*)", "Read(*)", "Write(*)", "Edit(*)", "Glob(*)", "Grep(*)", "WebFetch(*)", "WebSearch(*)"],
    "deny": []
  },
  "autoUpdaterStatus": "disabled",
  "spinnerTipsEnabled": false
}
EOF
    fi

    # Codex
    mkdir -p "${HOME}/.codex"
    if [[ ! -f "${HOME}/.codex/config.toml" ]]; then
        cat > "${HOME}/.codex/config.toml" << 'EOF'
model = "gpt-4.1"
approval_policy = "never"
sandbox_mode = "workspace-write"

[shell_environment_policy]
inherit = "all"
EOF
    fi

    # Gemini
    mkdir -p "${HOME}/.gemini"
    if [[ ! -f "${HOME}/.gemini/settings.json" ]]; then
        cat > "${HOME}/.gemini/settings.json" << 'EOF'
{
  "coreTools": ["read_file", "edit_file", "write_file", "run_shell_command", "search_files", "list_directory", "web_search"],
  "excludeTools": [],
  "theme": "Default (Dark)",
  "checkForUpdates": false,
  "sandbox": false,
  "yolo": false
}
EOF
    fi
}

create_config() {
    mkdir -p "$CONFIG_DIR"
    cat > "$CONFIG_FILE" << EOF
# AgentsMesh Runner — auto-generated dev config
server_url: "${BACKEND_URL}"
grpc_endpoint: "${GRPC_ENDPOINT}"
cert_file: "${CERTS_DIR}/runner.crt"
key_file: "${CERTS_DIR}/runner.key"
ca_file: "${CERTS_DIR}/ca.crt"
relay_base_url: "${RELAY_BASE_URL}"
node_id: "${RUNNER_NODE_ID}"
description: "Development Docker Runner (Bazel binary)"
org_slug: "${RUNNER_ORG_SLUG}"
max_concurrent_pods: ${MAX_CONCURRENT_PODS}
workspace: "/workspace"
workspace_root: "/workspace/repos"
worktrees_dir: "/workspace/worktrees"
base_branch: "main"
default_agent: "claude-code"
default_shell: "/bin/bash"
log_level: "debug"
EOF
}

main() {
    wait_for_backend
    generate_runner_cert
    init_ai_cli_configs
    create_config
    echo "启动 Runner (bazel-built binary)..."
    exec /usr/local/bin/agentsmesh-runner run
}

main "$@"
