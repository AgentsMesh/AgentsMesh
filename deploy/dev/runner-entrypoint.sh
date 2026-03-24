#!/bin/bash
# =============================================================================
# AgentsMesh Runner Docker Entrypoint
# =============================================================================
#
# 此脚本在 Runner 容器启动时执行：
# 1. 等待 Backend 服务就绪
# 2. 生成/复制 gRPC mTLS 客户端证书
# 3. 创建预配置的 config.yaml（使用 seed 数据中的 runner 信息）
# 4. 启动 Runner
#
# 环境变量：
#   BACKEND_URL       - Backend HTTP URL (for health check)
#   GRPC_ENDPOINT     - gRPC server endpoint (e.g., nginx:9443)
#   RUNNER_NODE_ID    - Runner 节点 ID (与 seed 数据匹配)
#   RUNNER_ORG_SLUG   - 组织 Slug (与 seed 数据匹配)
#   SSL_DIR           - SSL certificates directory (mounted from host)
#
# =============================================================================

set -e

# 默认配置（与 seed 数据匹配）
BACKEND_URL="${BACKEND_URL:-http://backend:8080}"
GRPC_ENDPOINT="${GRPC_ENDPOINT:-nginx:9443}"
RELAY_BASE_URL="${RELAY_BASE_URL:-}"
RUNNER_NODE_ID="${RUNNER_NODE_ID:-dev-runner}"
RUNNER_ORG_SLUG="${RUNNER_ORG_SLUG:-dev-org}"
MAX_CONCURRENT_PODS="${MAX_CONCURRENT_PODS:-10}"
SSL_DIR="${SSL_DIR:-/app/ssl}"

CONFIG_DIR="${HOME}/.agentsmesh"
CERTS_DIR="${CONFIG_DIR}/certs"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"

echo "========================================"
echo "  AgentsMesh Runner Entrypoint"
echo "========================================"
echo ""
echo "配置信息："
echo "  Backend URL:    $BACKEND_URL"
echo "  gRPC Endpoint:  $GRPC_ENDPOINT"
echo "  Relay Base URL: $RELAY_BASE_URL"
echo "  Node ID:        $RUNNER_NODE_ID"
echo "  Org Slug:       $RUNNER_ORG_SLUG"
echo "  Max Pods:       $MAX_CONCURRENT_PODS"
echo ""

# 等待 Backend 就绪
wait_for_backend() {
    echo "等待 Backend 服务就绪..."

    HEALTH_URL="${BACKEND_URL}/health"

    MAX_RETRIES=30
    RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if wget -q --spider "${HEALTH_URL}" 2>/dev/null; then
            echo "✓ Backend 服务就绪"
            return 0
        fi

        RETRY_COUNT=$((RETRY_COUNT + 1))
        echo "  等待 Backend... (${RETRY_COUNT}/${MAX_RETRIES})"
        sleep 2
    done

    echo "✗ Backend 服务启动超时"
    exit 1
}

# 生成 Runner 客户端证书 (dev 环境专用)
generate_runner_cert() {
    echo "生成 Runner 客户端证书..."

    mkdir -p "$CERTS_DIR"

    # 检查证书是否已存在
    if [ -f "$CERTS_DIR/runner.crt" ] && [ -f "$CERTS_DIR/runner.key" ]; then
        echo "✓ Runner 证书已存在"
        return 0
    fi

    # 检查 CA 证书是否存在
    if [ ! -f "$SSL_DIR/ca.crt" ] || [ ! -f "$SSL_DIR/ca.key" ]; then
        echo "✗ CA 证书未找到: $SSL_DIR"
        exit 1
    fi

    # 生成 Runner 私钥 (ECDSA P-256)
    openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 \
        -out "$CERTS_DIR/runner.key" 2>/dev/null

    # 生成 CSR (CN = node_id, O = org_slug)
    openssl req -new -key "$CERTS_DIR/runner.key" \
        -out "$CERTS_DIR/runner.csr" \
        -subj "/CN=${RUNNER_NODE_ID}/O=${RUNNER_ORG_SLUG}/OU=Runner" 2>/dev/null

    # 创建证书扩展配置
    cat > "$CERTS_DIR/runner_ext.cnf" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF

    # 用 CA 签发证书 (1 年有效期)
    # 注意: -CAserial 指向可写目录，避免只读挂载的问题
    openssl x509 -req -days 365 \
        -in "$CERTS_DIR/runner.csr" \
        -CA "$SSL_DIR/ca.crt" -CAkey "$SSL_DIR/ca.key" \
        -CAserial "$CERTS_DIR/ca.srl" -CAcreateserial \
        -out "$CERTS_DIR/runner.crt" \
        -extfile "$CERTS_DIR/runner_ext.cnf" 2>/dev/null

    # 复制 CA 证书
    cp "$SSL_DIR/ca.crt" "$CERTS_DIR/ca.crt"

    # 清理临时文件
    rm -f "$CERTS_DIR/runner.csr" "$CERTS_DIR/runner_ext.cnf"

    # 设置权限
    chmod 600 "$CERTS_DIR/runner.key"
    chmod 644 "$CERTS_DIR/runner.crt" "$CERTS_DIR/ca.crt"

    echo "✓ Runner 证书生成完成"
}

# 初始化 AI CLI 配置文件
init_ai_cli_configs() {
    echo "初始化 AI CLI 配置文件..."

    # Claude Code 配置
    # 注意: hasCompletedOnboarding 必须放在 ~/.claude.json 中才能跳过 onboarding
    # 为了持久化，我们把实际文件存在 ~/.claude/claude.json，然后创建符号链接到 ~/.claude.json
    CLAUDE_CONFIG_DIR="${HOME}/.claude"
    CLAUDE_JSON_ACTUAL="${CLAUDE_CONFIG_DIR}/claude.json"
    CLAUDE_JSON_LINK="${HOME}/.claude.json"
    CLAUDE_SETTINGS="${CLAUDE_CONFIG_DIR}/settings.json"

    mkdir -p "$CLAUDE_CONFIG_DIR"

    # 创建 ~/.claude/claude.json (实际存储位置，会被 volume 持久化)
    if [ ! -f "$CLAUDE_JSON_ACTUAL" ]; then
        cat > "$CLAUDE_JSON_ACTUAL" << 'EOF'
{
  "hasCompletedOnboarding": true,
  "theme": "dark",
  "autoUpdaterStatus": "disabled",
  "shiftEnterKeyBindingInstalled": true
}
EOF
        echo "  ✓ Claude Code claude.json 已初始化"
    else
        echo "  - Claude Code claude.json 已存在"
    fi

    # 创建符号链接 ~/.claude.json -> ~/.claude/claude.json
    if [ ! -L "$CLAUDE_JSON_LINK" ]; then
        rm -f "$CLAUDE_JSON_LINK"  # 删除可能存在的普通文件
        ln -s "$CLAUDE_JSON_ACTUAL" "$CLAUDE_JSON_LINK"
        echo "  ✓ Claude Code ~/.claude.json 符号链接已创建"
    else
        echo "  - Claude Code ~/.claude.json 符号链接已存在"
    fi

    # 创建 ~/.claude/settings.json (工具权限配置)
    if [ ! -f "$CLAUDE_SETTINGS" ]; then
        cat > "$CLAUDE_SETTINGS" << 'EOF'
{
  "permissions": {
    "allow": [
      "Bash(*)",
      "Read(*)",
      "Write(*)",
      "Edit(*)",
      "Glob(*)",
      "Grep(*)",
      "WebFetch(*)",
      "WebSearch(*)"
    ],
    "deny": []
  },
  "autoUpdaterStatus": "disabled",
  "spinnerTipsEnabled": false
}
EOF
        echo "  ✓ Claude Code settings.json 已初始化"
    else
        echo "  - Claude Code settings.json 已存在"
    fi

    # OpenAI Codex 配置
    CODEX_CONFIG_DIR="${HOME}/.codex"
    CODEX_CONFIG="${CODEX_CONFIG_DIR}/config.toml"
    if [ ! -f "$CODEX_CONFIG" ]; then
        mkdir -p "$CODEX_CONFIG_DIR"
        cat > "$CODEX_CONFIG" << 'EOF'
# Codex CLI Configuration
# Reference: https://developers.openai.com/codex/config-reference/

# Model settings
model = "gpt-4.1"

# Approval policy: "on-request", "never", "untrusted"
# For automated/headless mode, use "never" to skip manual approvals
approval_policy = "never"

# Sandbox mode: "read-only", "workspace-write", "danger-full-access"
# For development, allow full workspace write access
sandbox_mode = "workspace-write"

# Disable notifications for headless mode (notify is an array<string> command)
# notify = ["notify-send"]

# Shell environment policy
[shell_environment_policy]
inherit = "all"
EOF
        echo "  ✓ OpenAI Codex 配置已初始化"
    else
        echo "  - OpenAI Codex 配置已存在"
    fi

    # Gemini CLI 配置
    GEMINI_CONFIG_DIR="${HOME}/.gemini"
    GEMINI_SETTINGS="${GEMINI_CONFIG_DIR}/settings.json"
    if [ ! -f "$GEMINI_SETTINGS" ]; then
        mkdir -p "$GEMINI_CONFIG_DIR"
        cat > "$GEMINI_SETTINGS" << 'EOF'
{
  "coreTools": [
    "read_file",
    "edit_file",
    "write_file",
    "run_shell_command",
    "search_files",
    "list_directory",
    "web_search"
  ],
  "excludeTools": [],
  "theme": "Default (Dark)",
  "checkForUpdates": false,
  "sandbox": false,
  "yolo": false
}
EOF
        echo "  ✓ Gemini CLI 配置已初始化"
    else
        echo "  - Gemini CLI 配置已存在"
    fi

    echo "✓ AI CLI 配置初始化完成"
}

# 创建配置文件
create_config() {
    echo "创建 Runner 配置文件..."

    mkdir -p "$CONFIG_DIR"

    cat > "$CONFIG_FILE" << EOF
# AgentsMesh Runner Configuration
# Auto-generated for Docker development environment

# Server connection (for REST API calls like certificate renewal)
server_url: "${BACKEND_URL}"

# gRPC + mTLS connection
grpc_endpoint: "${GRPC_ENDPOINT}"
cert_file: "${CERTS_DIR}/runner.crt"
key_file: "${CERTS_DIR}/runner.key"
ca_file: "${CERTS_DIR}/ca.crt"

# Relay URL override (Docker: rewrite external relay URL to Docker-internal Traefik)
relay_base_url: "${RELAY_BASE_URL}"

# Runner identification
node_id: "${RUNNER_NODE_ID}"
description: "Development Docker Runner"

# Organization
org_slug: "${RUNNER_ORG_SLUG}"

# Capacity
max_concurrent_pods: ${MAX_CONCURRENT_PODS}

# Workspace settings
workspace: "/workspace"
workspace_root: "/workspace/repos"

# Sandbox settings (worktree plugin)
worktrees_dir: "/workspace/worktrees"
base_branch: "main"

# Agent settings
default_agent: "claude-code"
default_shell: "/bin/bash"

# Logging
log_level: "debug"
EOF

    echo "✓ 配置文件已创建: $CONFIG_FILE"
}

# 显示配置内容
show_config() {
    echo ""
    echo "配置文件内容："
    echo "----------------------------------------"
    cat "$CONFIG_FILE"
    echo "----------------------------------------"
    echo ""
}

# 启动 Runner
start_runner() {
    echo "启动 Runner..."
    echo ""

    # 使用 Air 进行热重载开发
    if command -v air &> /dev/null; then
        echo "使用 Air 热重载模式..."
        exec air -c .air.toml
    else
        # 直接运行 go run
        echo "使用 go run 模式..."
        exec go run ./cmd/runner run
    fi
}

# 主流程
main() {
    wait_for_backend
    generate_runner_cert
    init_ai_cli_configs
    create_config

    if [ "${DEBUG:-false}" = "true" ]; then
        show_config
    fi

    start_runner
}

main "$@"
