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
echo "  Backend URL:   $BACKEND_URL"
echo "  gRPC Endpoint: $GRPC_ENDPOINT"
echo "  Node ID:       $RUNNER_NODE_ID"
echo "  Org Slug:      $RUNNER_ORG_SLUG"
echo "  Max Pods:      $MAX_CONCURRENT_PODS"
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

    # 生成 CSR (CN = node_id)
    openssl req -new -key "$CERTS_DIR/runner.key" \
        -out "$CERTS_DIR/runner.csr" \
        -subj "/CN=${RUNNER_NODE_ID}/O=AgentsMesh/OU=Runner" 2>/dev/null

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
    create_config

    if [ "${DEBUG:-false}" = "true" ]; then
        show_config
    fi

    start_runner
}

main "$@"
