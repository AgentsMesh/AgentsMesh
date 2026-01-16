#!/bin/bash
# =============================================================================
# AgentsMesh Runner Docker Entrypoint
# =============================================================================
#
# 此脚本在 Runner 容器启动时执行：
# 1. 等待 Backend 服务就绪
# 2. 创建预配置的 config.yaml（使用 seed 数据中的 runner 信息）
# 3. 启动 Runner
#
# 环境变量：
#   BACKEND_URL       - Backend WebSocket URL
#   RUNNER_AUTH_TOKEN - Runner 认证令牌 (与 seed 数据匹配)
#   RUNNER_NODE_ID    - Runner 节点 ID (与 seed 数据匹配)
#   RUNNER_ORG_SLUG   - 组织 Slug (与 seed 数据匹配)
#
# =============================================================================

set -e

# 默认配置（与 seed 数据匹配）
BACKEND_URL="${BACKEND_URL:-ws://backend:8080}"
RUNNER_AUTH_TOKEN="${RUNNER_AUTH_TOKEN:-dev-runner-auth-token}"
RUNNER_NODE_ID="${RUNNER_NODE_ID:-dev-runner}"
RUNNER_ORG_SLUG="${RUNNER_ORG_SLUG:-dev-org}"
MAX_CONCURRENT_PODS="${MAX_CONCURRENT_PODS:-10}"

CONFIG_DIR="${HOME}/.agentsmesh"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"

echo "========================================"
echo "  AgentsMesh Runner Entrypoint"
echo "========================================"
echo ""
echo "配置信息："
echo "  Backend URL:  $BACKEND_URL"
echo "  Node ID:      $RUNNER_NODE_ID"
echo "  Org Slug:     $RUNNER_ORG_SLUG"
echo "  Max Pods:     $MAX_CONCURRENT_PODS"
echo ""

# 等待 Backend 就绪
wait_for_backend() {
    echo "等待 Backend 服务就绪..."

    # 从 ws:// 转换为 http:// 用于健康检查
    HTTP_URL=$(echo "$BACKEND_URL" | sed 's|^ws://|http://|' | sed 's|^wss://|https://|')
    HEALTH_URL="${HTTP_URL}/health"

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

# 创建配置文件
create_config() {
    echo "创建 Runner 配置文件..."

    mkdir -p "$CONFIG_DIR"

    cat > "$CONFIG_FILE" << EOF
# AgentsMesh Runner Configuration
# Auto-generated for Docker development environment

# Server connection
server_url: "${BACKEND_URL}"

# Runner identification
node_id: "${RUNNER_NODE_ID}"
description: "Development Docker Runner"

# Authentication (matches seed data)
auth_token: "${RUNNER_AUTH_TOKEN}"

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
    create_config

    if [ "${DEBUG:-false}" = "true" ]; then
        show_config
    fi

    start_runner
}

main "$@"
