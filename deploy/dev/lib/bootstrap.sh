# shellcheck shell=bash
# bootstrap.sh — once-per-environment data setup.
#
# After docker compose up, this module:
#   - waits for postgres + gitea readiness
#   - runs golang-migrate via the docker-compose `migrate` profile
#   - seeds users + LemonSqueezy variant ids
#   - registers the runner SSH key with Gitea
#   - writes ~/.ssh/config so host-side `git@gitea:...` resolves
# Re-runs are idempotent (existing data → skip).

# Generic docker-exec health probe. Polls `docker exec $container $check_cmd`
# at 2s intervals up to 240 attempts (8min) — cold CI pulls + first-start
# init can run well past 60s.
wait_for_service() {
    local container="$1"
    local check_cmd="$2"
    local max_retries=240

    for ((i=1; i<=max_retries; i++)); do
        if docker exec "$container" $check_cmd &>/dev/null; then
            return 0
        fi
        sleep 2
    done
    return 1
}

# golang-migrate via the docker compose `migrate` oneshot service.
# Detects + repairs dirty state; refuses to force a fresh DB so a broken
# migration surfaces loudly rather than getting masked.
run_migrations() {
    local db_url="postgres://agentsmesh:agentsmesh_dev@localhost:5432/agentsmesh?sslmode=disable"
    local compose_run="docker compose run --rm --no-deps migrate"

    info "执行数据库迁移 (docker oneshot migrate)..."

    local migration_output
    migration_output=$($compose_run -database "$db_url" version 2>&1) || true
    if echo "$migration_output" | grep -q "dirty"; then
        warn "检测到迁移状态为 dirty，尝试修复..."
        local dirty_version
        dirty_version=$(echo "$migration_output" | grep -oE '[0-9]+' | head -1)
        if [[ -n "$dirty_version" ]]; then
            $compose_run -database "$db_url" force "$dirty_version" >/dev/null 2>&1 || true
            success "已修复 dirty 状态 (version: $dirty_version)"
        fi
    fi

    local migrate_result
    migrate_result=$($compose_run -database "$db_url" up 2>&1) || true

    if echo "$migrate_result" | grep -q "no change"; then
        info "数据库已是最新版本"
    elif echo "$migrate_result" | grep -q "^error"; then
        error "迁移失败:"
        echo "$migrate_result" | sed 's/^/    /'
        local current_version latest_version
        current_version=$($compose_run -database "$db_url" version 2>&1 | grep -oE '^[0-9]+' || echo "0")
        latest_version=$(ls -1 "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | \
            sed 's/.*\/\([0-9]*\)_.*/\1/' | sort -n | tail -1)

        if [[ -n "$latest_version" && "$current_version" != "0" && "$current_version" != "$latest_version" ]]; then
            warn "已有部分迁移应用 (version=$current_version)，强制设置到 $latest_version"
            $compose_run -database "$db_url" force "$latest_version" >/dev/null 2>&1 || true
        else
            error "Fresh database — refusing to force version. Fix the migration and rerun."
            return 1
        fi
    else
        success "数据库迁移完成"
    fi

    local final_version
    final_version=$($compose_run -database "$db_url" version 2>&1 | head -1)
    info "当前迁移版本: $final_version"
}

init_seed() {
    local pg_container="$1"

    local user_exists
    user_exists=$(docker exec "$pg_container" psql -U agentsmesh -d agentsmesh -t -c \
        "SELECT COUNT(*) FROM users WHERE email = 'dev@agentsmesh.local'" 2>/dev/null | tr -d ' ')

    if [[ "$user_exists" -gt 0 ]]; then
        info "Seed 数据已存在，跳过"
        return 0
    fi

    info "初始化 seed 数据..."
    docker exec -i "$pg_container" psql -U agentsmesh -d agentsmesh < "$SEED_FILE" &>/dev/null

    if [[ -f "$LEMONSQUEEZY_SEED_FILE" ]]; then
        info "配置 LemonSqueezy Variant IDs..."
        docker exec -i "$pg_container" psql -U agentsmesh -d agentsmesh < "$LEMONSQUEEZY_SEED_FILE" &>/dev/null
    fi

    success "Seed 数据初始化完成"
}

# Gitea-side bootstrap: admin user + dev-org + 2 demo repos + register the
# runner SSH public key as a deploy key. Delegated to gitea/init-gitea.sh
# so the dev.sh main flow stays declarative.
init_gitea() {
    local gitea_container="${COMPOSE_PROJECT_NAME}-gitea-1"
    source "$ENV_FILE"
    local gitea_port="${GITEA_HTTP_PORT:-3001}"

    info "等待 Gitea 就绪..."
    local max_retries=30
    for ((i=1; i<=max_retries; i++)); do
        if curl -s "http://localhost:${gitea_port}/api/v1/version" &>/dev/null; then
            break
        fi
        if [[ $i -eq $max_retries ]]; then
            warn "Gitea 启动超时，跳过初始化"
            return 0
        fi
        sleep 2
    done
    success "Gitea 已就绪"

    "$SCRIPT_DIR/gitea/init-gitea.sh" "$gitea_container" "$gitea_port"
}

# Configure ~/.ssh/config so host-side `git@gitea:org/repo.git` resolves.
# Inside docker, `gitea` is a service-DNS hostname; on the host it would
# fail with "nodename not known". We add a managed Host block that maps
# gitea → 127.0.0.1:GITEA_SSH_PORT. Idempotent — old block is stripped
# before the new one is written, so the port is always current.
setup_gitea_ssh_config() {
    source "$ENV_FILE"
    local ssh_dir="$HOME/.ssh"
    local ssh_config="$ssh_dir/config"
    local gitea_ssh_port="${GITEA_SSH_PORT:-2222}"
    local identity_file="$SCRIPT_DIR/runner-ssh/id_ed25519"
    local marker_start="# BEGIN AgentsMesh dev gitea"
    local marker_end="# END AgentsMesh dev gitea"

    mkdir -p "$ssh_dir"
    chmod 700 "$ssh_dir"
    [[ -f "$ssh_config" ]] || touch "$ssh_config"
    chmod 600 "$ssh_config"

    local tmp
    tmp=$(mktemp)
    awk "/^${marker_start}$/,/^${marker_end}$/{next} {print}" "$ssh_config" > "$tmp"
    cat "$tmp" > "$ssh_config"
    rm -f "$tmp"

    cat >> "$ssh_config" << EOF

${marker_start}
Host gitea
    HostName 127.0.0.1
    Port ${gitea_ssh_port}
    IdentityFile ${identity_file}
    IdentitiesOnly yes
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
${marker_end}
EOF

    success "SSH config: git@gitea:... → 127.0.0.1:${gitea_ssh_port} (key: runner-ssh/id_ed25519)"
}
