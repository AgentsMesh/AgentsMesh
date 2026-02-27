#!/bin/bash
# =============================================================================
# Gitea 初始化脚本
# =============================================================================
# 在 Gitea 容器启动后调用，完成以下工作：
#   1. 创建管理员用户
#   2. 创建组织 (dev-org)
#   3. 创建 seed 仓库 (demo-webapp, demo-api)
#   4. 推送预置代码
#   5. 注册 Runner SSH deploy key
#
# 使用方法: ./init-gitea.sh <gitea-container-name> <gitea-http-port>
# =============================================================================

set -e

GITEA_CONTAINER="$1"
GITEA_HTTP_PORT="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ADMIN_USER="gitea-admin"
ADMIN_PASS="gitea-admin-123"
ADMIN_EMAIL="admin@gitea.local"
ORG_NAME="dev-org"
GITEA_API="http://localhost:${GITEA_HTTP_PORT}/api/v1"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'
info() { echo -e "${BLUE}  [gitea]${NC} $1"; }
success() { echo -e "${GREEN}  [gitea]${NC} $1"; }
error() { echo -e "${RED}  [gitea]${NC} $1"; }

# Wait for Gitea API to be ready
wait_for_gitea() {
    local max_retries=30
    for ((i=1; i<=max_retries; i++)); do
        if curl -s "${GITEA_API}/version" &>/dev/null; then
            return 0
        fi
        sleep 2
    done
    error "Gitea API not ready after ${max_retries} retries"
    return 1
}

# Create admin user via CLI (most reliable method)
create_admin_user() {
    # Check if admin user already exists
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" \
        -u "${ADMIN_USER}:${ADMIN_PASS}" \
        "${GITEA_API}/user")

    if [[ "$status" == "200" ]]; then
        info "Admin user already exists"
        return 0
    fi

    info "Creating admin user..."
    # Must run as 'git' user inside the container (Gitea refuses to run as root)
    docker exec -u git "$GITEA_CONTAINER" gitea admin user create \
        --admin \
        --username "$ADMIN_USER" \
        --password "$ADMIN_PASS" \
        --email "$ADMIN_EMAIL" \
        --must-change-password=false 2>/dev/null || true
    success "Admin user created"
}

# API helper with Basic Auth
api() {
    local method="$1"
    local path="$2"
    local data="$3"

    local args=(-s -X "$method" -H "Content-Type: application/json")
    args+=(-u "${ADMIN_USER}:${ADMIN_PASS}")

    if [[ -n "$data" ]]; then
        args+=(-d "$data")
    fi

    curl "${args[@]}" "${GITEA_API}${path}"
}

# Create organization
create_org() {
    # Check if org exists
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" \
        -u "${ADMIN_USER}:${ADMIN_PASS}" \
        "${GITEA_API}/orgs/${ORG_NAME}")

    if [[ "$status" == "200" ]]; then
        info "Organization '${ORG_NAME}' already exists"
        return 0
    fi

    info "Creating organization '${ORG_NAME}'..."
    api POST "/orgs" "{\"username\":\"${ORG_NAME}\",\"visibility\":\"public\"}" > /dev/null
    success "Organization created"
}

# Create a repo and push seed code
create_and_push_repo() {
    local repo_name="$1"
    local source_dir="$2"

    # Check if repo exists
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" \
        -u "${ADMIN_USER}:${ADMIN_PASS}" \
        "${GITEA_API}/repos/${ORG_NAME}/${repo_name}")

    if [[ "$status" == "200" ]]; then
        info "Repository '${ORG_NAME}/${repo_name}' already exists"
        return 0
    fi

    info "Creating repository '${ORG_NAME}/${repo_name}'..."
    api POST "/orgs/${ORG_NAME}/repos" \
        "{\"name\":\"${repo_name}\",\"default_branch\":\"main\",\"auto_init\":false}" > /dev/null

    # Push seed code via HTTPS (using admin credentials)
    local tmp_dir
    tmp_dir=$(mktemp -d)
    cp -r "${source_dir}/"* "$tmp_dir/"

    (
        cd "$tmp_dir"
        git init -b main > /dev/null 2>&1
        git add . > /dev/null 2>&1
        git commit -m "Initial commit: seed project" > /dev/null 2>&1
        git remote add origin \
            "http://${ADMIN_USER}:${ADMIN_PASS}@localhost:${GITEA_HTTP_PORT}/${ORG_NAME}/${repo_name}.git"
        git push -u origin main > /dev/null 2>&1
    )

    rm -rf "$tmp_dir"
    success "Repository '${ORG_NAME}/${repo_name}' created and seeded"
}

# Add SSH deploy key to a repo
add_deploy_key() {
    local repo_name="$1"
    local key_file="$2"
    local key_title="runner-deploy-key"

    # Check if deploy key already exists
    local existing
    existing=$(curl -s -u "${ADMIN_USER}:${ADMIN_PASS}" \
        "${GITEA_API}/repos/${ORG_NAME}/${repo_name}/keys" | grep -c "$key_title" || true)

    if [[ "$existing" -gt 0 ]]; then
        info "Deploy key already exists for '${repo_name}'"
        return 0
    fi

    local pub_key
    pub_key=$(cat "$key_file")

    info "Adding deploy key to '${ORG_NAME}/${repo_name}'..."
    api POST "/repos/${ORG_NAME}/${repo_name}/keys" \
        "{\"title\":\"${key_title}\",\"key\":\"${pub_key}\",\"read_only\":false}" > /dev/null
    success "Deploy key added to '${repo_name}'"
}

# =============================================================================
# Main
# =============================================================================

info "Initializing Gitea..."

# 1. Wait for API
wait_for_gitea

# 2. Create admin user
create_admin_user

# 3. Create organization
create_org

# 4. Create and seed repositories
create_and_push_repo "demo-webapp" "${SCRIPT_DIR}/repos/demo-webapp"
create_and_push_repo "demo-api" "${SCRIPT_DIR}/repos/demo-api"

# 5. Add deploy key (same SSH key used by Runner)
SSH_PUB_KEY="${SCRIPT_DIR}/../runner-ssh/id_ed25519.pub"
if [[ -f "$SSH_PUB_KEY" ]]; then
    add_deploy_key "demo-webapp" "$SSH_PUB_KEY"
    add_deploy_key "demo-api" "$SSH_PUB_KEY"
else
    error "SSH public key not found: $SSH_PUB_KEY"
fi

success "Gitea initialization complete!"
info "  Admin:  http://localhost:${GITEA_HTTP_PORT} (${ADMIN_USER} / ${ADMIN_PASS})"
info "  Repos:  ${ORG_NAME}/demo-webapp, ${ORG_NAME}/demo-api"
