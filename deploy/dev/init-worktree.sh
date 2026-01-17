#!/bin/bash
# =============================================================================
# AgentsMesh 开发环境一键初始化脚本
# =============================================================================
#
# 一键启动完整的开发环境，包括：
#   1. 生成 worktree 隔离的 .env 配置
#   2. 启动所有 Docker 服务
#   3. 执行数据库迁移
#   4. 初始化 seed 数据
#
# 使用方法：
#   ./init-worktree.sh         # 一键启动开发环境
#   ./init-worktree.sh --clean # 清理并重建（失败时使用）
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
MIGRATIONS_DIR="$SCRIPT_DIR/../../backend/migrations"
SEED_FILE="$SCRIPT_DIR/seed/seed.sql"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 获取 worktree 名称
get_worktree_name() {
    local git_dir
    git_dir=$(git rev-parse --git-dir 2>/dev/null)

    if [[ "$git_dir" == *".git/worktrees/"* ]]; then
        # worktree 的 git-dir 格式: /path/to/repo/.git/worktrees/<worktree-name>
        basename "$git_dir"
    else
        git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main"
    fi | sed 's/[^a-zA-Z0-9-]/-/g' | tr '[:upper:]' '[:lower:]'
}

# 计算端口偏移量
calculate_port_offset() {
    local name="$1"
    if [[ "$name" == "main" || "$name" == "master" ]]; then
        echo 0
    else
        local hash
        if command -v md5sum &>/dev/null; then
            hash=$(echo -n "$name" | md5sum | cut -c1-4)
        else
            hash=$(echo -n "$name" | md5 | cut -c1-4)
        fi
        echo $(( (16#$hash % 99) + 1 ))
    fi
}

# 生成 SSL 证书 (gRPC + mTLS)
generate_ssl_certs() {
    local ssl_dir="$SCRIPT_DIR/ssl"
    if [ -f "$ssl_dir/ca.crt" ] && [ -f "$ssl_dir/ca.key" ]; then
        info "SSL 证书已存在"
        return 0
    fi

    info "生成 SSL 证书 (gRPC + mTLS)..."
    "$SCRIPT_DIR/generate-dev-certs.sh" > /dev/null 2>&1
    success "SSL 证书生成完成"
}

# 生成 .env 配置
generate_env() {
    local worktree_name=$(get_worktree_name)
    local offset=$(calculate_port_offset "$worktree_name")
    local project_name="agentsmesh-${worktree_name}"

    cat > "$ENV_FILE" << EOF
# AgentsMesh Dev Environment - Auto-generated
# Worktree: $worktree_name | Offset: $offset

COMPOSE_PROJECT_NAME=$project_name

# Ports
HTTP_PORT=$((80 + offset * 100))
GRPC_PORT=$((9443 + offset * 100))
POSTGRES_PORT=$((5432 + offset * 100))
REDIS_PORT=$((6379 + offset * 100))
MINIO_API_PORT=$((9000 + offset * 100))
MINIO_CONSOLE_PORT=$((9001 + offset * 100))
ADMINER_PORT=$((8081 + offset * 100))

# Credentials
POSTGRES_PASSWORD=agentsmesh_dev
JWT_SECRET=dev-jwt-secret-change-in-production
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin

# OAuth (optional)
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
EOF
    success "生成 .env 配置 (worktree: $worktree_name)"
}

# 等待服务就绪
wait_for_service() {
    local container="$1"
    local check_cmd="$2"
    local max_retries=30

    for ((i=1; i<=max_retries; i++)); do
        if docker exec "$container" $check_cmd &>/dev/null; then
            return 0
        fi
        sleep 2
    done
    return 1
}

# 执行数据库迁移
run_migrations() {
    local pg_container="$1"

    # 检查是否已有表
    local table_count
    table_count=$(docker exec "$pg_container" psql -U agentsmesh -d agentsmesh -t -c \
        "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'" 2>/dev/null | tr -d ' ')

    if [[ "$table_count" -gt 0 ]]; then
        info "数据库已初始化，跳过迁移"
        return 0
    fi

    info "执行数据库迁移..."
    for f in "$MIGRATIONS_DIR"/*.up.sql; do
        [[ -f "$f" ]] && docker exec -i "$pg_container" psql -U agentsmesh -d agentsmesh < "$f" &>/dev/null
    done
    success "数据库迁移完成"
}

# 初始化 seed 数据
init_seed() {
    local pg_container="$1"

    # 检查是否已有 seed 数据
    local user_exists
    user_exists=$(docker exec "$pg_container" psql -U agentsmesh -d agentsmesh -t -c \
        "SELECT COUNT(*) FROM users WHERE email = 'dev@agentsmesh.local'" 2>/dev/null | tr -d ' ')

    if [[ "$user_exists" -gt 0 ]]; then
        info "Seed 数据已存在，跳过"
        return 0
    fi

    info "初始化 seed 数据..."
    docker exec -i "$pg_container" psql -U agentsmesh -d agentsmesh < "$SEED_FILE" &>/dev/null
    success "Seed 数据初始化完成"
}

# 清理环境
clean() {
    if [[ -f "$ENV_FILE" ]]; then
        source "$ENV_FILE"
        info "清理环境: ${COMPOSE_PROJECT_NAME:-agentsmesh}..."
        cd "$SCRIPT_DIR"
        docker compose down -v --remove-orphans 2>/dev/null || true
        rm -f "$ENV_FILE"
        success "清理完成"
    else
        warn "环境未初始化，无需清理"
    fi
}

# 显示结果
show_result() {
    source "$ENV_FILE"
    echo ""
    echo "=========================================="
    echo "  AgentsMesh 开发环境已就绪!"
    echo "=========================================="
    echo ""
    echo "  访问地址:  http://localhost:$HTTP_PORT"
    echo ""
    echo "  测试账号:"
    echo "    Email:    dev@agentsmesh.local"
    echo "    Password: devpass123"
    echo ""
    echo "  其他服务:"
    echo "    Adminer:  http://localhost:$ADMINER_PORT"
    echo "    MinIO:    http://localhost:$MINIO_CONSOLE_PORT"
    echo "    gRPC:     grpcs://localhost:${GRPC_PORT:-9443} (mTLS)"
    echo ""
    echo "  停止: docker compose down"
    echo "  重建: ./init-worktree.sh --clean && ./init-worktree.sh"
    echo ""
}

# 主流程
main() {
    cd "$SCRIPT_DIR"

    # 处理 --clean 参数
    if [[ "${1:-}" == "--clean" || "${1:-}" == "-c" ]]; then
        clean
        exit 0
    fi

    # 显示帮助
    if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
        echo "用法: $0 [--clean]"
        echo ""
        echo "  无参数    一键启动开发环境"
        echo "  --clean   清理环境（失败时使用，然后重新运行）"
        exit 0
    fi

    echo ""
    echo "=========================================="
    echo "  AgentsMesh 开发环境初始化"
    echo "=========================================="
    echo ""

    # Step 1: 生成 SSL 证书
    generate_ssl_certs

    # Step 2: 生成配置
    generate_env
    source "$ENV_FILE"

    # Step 3: 启动服务
    info "启动 Docker 服务 (首次可能需要几分钟)..."
    docker compose up -d --build --quiet-pull 2>&1 | grep -v "^#" | grep -v "^\[" | grep -v "^$" || true
    success "Docker 服务已启动"

    # Step 4: 等待 PostgreSQL
    local pg_container="${COMPOSE_PROJECT_NAME}-postgres-1"
    info "等待 PostgreSQL 就绪..."
    if ! wait_for_service "$pg_container" "pg_isready -U agentsmesh"; then
        error "PostgreSQL 启动超时"
        exit 1
    fi
    success "PostgreSQL 已就绪"

    # Step 5: 执行迁移
    run_migrations "$pg_container"

    # Step 6: 初始化 seed
    init_seed "$pg_container"

    # Step 7: 修复 workspace 权限 (runner 容器)
    local runner_container="${COMPOSE_PROJECT_NAME}-runner-1"
    docker exec -u root "$runner_container" chown -R runner:runner /workspace 2>/dev/null || true

    # 显示结果
    show_result
}

main "$@"
