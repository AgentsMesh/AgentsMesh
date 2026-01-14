#!/bin/bash
# =============================================================================
# AgentMesh Seed Data 初始化脚本
# =============================================================================
#
# 此脚本在数据库迁移后执行，创建开发环境所需的初始数据。
#
# 使用方法：
#   ./init-seed.sh              # 应用 seed 数据
#   ./init-seed.sh --reset      # 重置数据库并重新应用 seed
#   ./init-seed.sh --status     # 检查 seed 数据状态
#
# 环境变量：
#   POSTGRES_HOST     - PostgreSQL 主机 (默认: localhost)
#   POSTGRES_PORT     - PostgreSQL 端口 (默认: 5432)
#   POSTGRES_USER     - PostgreSQL 用户 (默认: agentmesh)
#   POSTGRES_PASSWORD - PostgreSQL 密码 (默认: agentmesh_dev)
#   POSTGRES_DB       - PostgreSQL 数据库 (默认: agentmesh)
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SEED_FILE="$SCRIPT_DIR/seed/seed.sql"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 加载 .env 文件
load_env() {
    if [[ -f "$SCRIPT_DIR/.env" ]]; then
        source "$SCRIPT_DIR/.env"
    fi
}

# 获取数据库连接参数
get_db_params() {
    load_env

    DB_HOST="${POSTGRES_HOST:-localhost}"
    DB_PORT="${POSTGRES_PORT:-5432}"
    DB_USER="${POSTGRES_USER:-agentmesh}"
    DB_PASSWORD="${POSTGRES_PASSWORD:-agentmesh_dev}"
    DB_NAME="${POSTGRES_DB:-agentmesh}"

    export PGPASSWORD="$DB_PASSWORD"
}

# 检查 PostgreSQL 连接
check_connection() {
    info "检查数据库连接..."
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; then
        error "无法连接到数据库"
        echo ""
        echo "请确保："
        echo "  1. PostgreSQL 服务正在运行"
        echo "  2. 数据库 '$DB_NAME' 已创建"
        echo "  3. 连接参数正确："
        echo "     - Host: $DB_HOST"
        echo "     - Port: $DB_PORT"
        echo "     - User: $DB_USER"
        echo "     - Database: $DB_NAME"
        exit 1
    fi
    success "数据库连接成功"
}

# 检查迁移状态
check_migrations() {
    info "检查数据库迁移状态..."

    # 检查 users 表是否存在
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1 FROM users LIMIT 1" &>/dev/null; then
        error "数据库表不存在，请先运行迁移"
        echo ""
        echo "运行迁移命令："
        echo "  cd backend && ./scripts/migrate.sh up"
        exit 1
    fi
    success "数据库迁移已完成"
}

# 应用 seed 数据
apply_seed() {
    info "应用 seed 数据..."

    if [[ ! -f "$SEED_FILE" ]]; then
        error "Seed 文件不存在: $SEED_FILE"
        exit 1
    fi

    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SEED_FILE"; then
        success "Seed 数据应用成功!"
        echo ""
        echo "测试账号信息："
        echo "  Email:    dev@agentmesh.local"
        echo "  Password: devpass123"
        echo "  Org:      dev-org"
        echo ""
        echo "Runner 信息："
        echo "  Node ID:    dev-runner"
        echo "  Auth Token: dev-runner-auth-token"
    else
        error "Seed 数据应用失败"
        exit 1
    fi
}

# 检查 seed 数据状态
check_status() {
    info "检查 seed 数据状态..."
    echo ""

    # 检查用户
    echo "用户:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
        "SELECT id, email, username, is_active FROM users WHERE email = 'dev@agentmesh.local'" \
        --tuples-only 2>/dev/null | while read line; do
        if [[ -n "$line" ]]; then
            echo "  ✓ dev@agentmesh.local 已存在"
        fi
    done

    # 检查组织
    echo "组织:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
        "SELECT id, name, slug FROM organizations WHERE slug = 'dev-org'" \
        --tuples-only 2>/dev/null | while read line; do
        if [[ -n "$line" ]]; then
            echo "  ✓ dev-org 已存在"
        fi
    done

    # 检查 Runner
    echo "Runner:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
        "SELECT id, node_id, status FROM runners WHERE node_id = 'dev-runner'" \
        --tuples-only 2>/dev/null | while read line; do
        if [[ -n "$line" ]]; then
            echo "  ✓ dev-runner 已存在"
        fi
    done

    # 检查 Ticket
    echo "Tickets:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
        "SELECT COUNT(*) FROM tickets WHERE organization_id = (SELECT id FROM organizations WHERE slug = 'dev-org')" \
        --tuples-only 2>/dev/null | while read count; do
        echo "  共 ${count// /} 个 Ticket"
    done

    echo ""
}

# 重置数据库
reset_database() {
    warn "即将重置数据库中的 seed 数据..."
    echo ""
    read -p "确认重置？此操作将删除测试数据 [y/N] " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        info "取消重置"
        exit 0
    fi

    info "删除现有 seed 数据..."

    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'EOF'
-- 删除 seed 数据（按依赖顺序）
DELETE FROM tickets WHERE organization_id IN (SELECT id FROM organizations WHERE slug = 'dev-org');
DELETE FROM runners WHERE organization_id IN (SELECT id FROM organizations WHERE slug = 'dev-org');
DELETE FROM runner_registration_tokens WHERE organization_id IN (SELECT id FROM organizations WHERE slug = 'dev-org');
DELETE FROM organization_members WHERE organization_id IN (SELECT id FROM organizations WHERE slug = 'dev-org');
DELETE FROM organizations WHERE slug = 'dev-org';
DELETE FROM users WHERE email = 'dev@agentmesh.local';
EOF

    success "现有 seed 数据已删除"
    apply_seed
}

# 显示帮助
show_help() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --status    检查 seed 数据状态"
    echo "  --reset     重置并重新应用 seed 数据"
    echo "  --help, -h  显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0            # 应用 seed 数据"
    echo "  $0 --status   # 检查状态"
    echo "  $0 --reset    # 重置数据"
    echo ""
    echo "环境变量:"
    echo "  POSTGRES_HOST     PostgreSQL 主机 (默认: localhost)"
    echo "  POSTGRES_PORT     PostgreSQL 端口 (默认: 5432)"
    echo "  POSTGRES_USER     PostgreSQL 用户 (默认: agentmesh)"
    echo "  POSTGRES_PASSWORD PostgreSQL 密码 (默认: agentmesh_dev)"
    echo "  POSTGRES_DB       PostgreSQL 数据库 (默认: agentmesh)"
}

# 主函数
main() {
    get_db_params

    case "${1:-}" in
        --status|-s)
            check_connection
            check_status
            ;;
        --reset|-r)
            check_connection
            check_migrations
            reset_database
            ;;
        --help|-h)
            show_help
            ;;
        *)
            check_connection
            check_migrations
            apply_seed
            ;;
    esac
}

main "$@"
