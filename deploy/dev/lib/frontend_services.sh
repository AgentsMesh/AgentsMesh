# shellcheck shell=bash
# Frontend dependency checks and isolated Bazel devserver lifecycle.

# Reusable lockfile-driven pnpm install: skips if node_modules is in sync
# with pnpm-lock.yaml (md5 fingerprint), reinstalls otherwise. Returns
# non-zero on install failure so callers can decide fail-vs-skip.
_install_root_deps_if_needed() {
    local context="$1"            # human label for logs ("前端依赖" / "Admin Console 依赖")
    local stale_cache_dir="$2"    # .next/cache to wipe on reinstall
    local root_dir="$SCRIPT_DIR/../.."
    local lockfile="$root_dir/pnpm-lock.yaml"
    local lockfile_hash_file="$root_dir/node_modules/.pnpm-lock-hash"
    local current_hash="" cached_hash=""
    [[ -f "$lockfile" ]] && current_hash=$(md5 -q "$lockfile" 2>/dev/null || md5sum "$lockfile" | cut -d' ' -f1)
    [[ -f "$lockfile_hash_file" ]] && cached_hash=$(cat "$lockfile_hash_file")

    if [[ -d "$root_dir/node_modules" && "$current_hash" == "$cached_hash" ]]; then
        return 0
    fi

    info "安装 ${context}（根 workspace）..."
    if ! (cd "$root_dir" && pnpm install --frozen-lockfile); then
        error "${context} 安装失败"
        return 1
    fi
    echo "$current_hash" > "$lockfile_hash_file"
    rm -rf "$stale_cache_dir"
    success "${context} 安装完成"
}

# Each long-running next_dev process needs its own Bazel server. Reusing the
# workspace's default output base means starting Admin, building another target,
# or running E2E terminates the already-running Web devserver. The worktree name
# scopes these paths across concurrent checkouts; /tmp keeps them outside every
# Bazel workspace and lets the OS reclaim cold caches.
_frontend_bazel_output_base() {
    local frontend="$1"
    local worktree_name
    worktree_name=$(get_worktree_name)
    local root="${AGENTSMESH_DEV_BAZEL_ROOT:-/tmp}"
    echo "${root%/}/${worktree_name}-bazel-${frontend}"
}

# Bazel output trees contain read-only directories. Cache cleanup is
# best-effort because it must never abort the remaining dev teardown.
_remove_frontend_bazel_output_base() {
    local output_base="$1"

    if rm -rf -- "$output_base" 2>/dev/null; then
        return 0
    fi

    warn "Bazel cache removal failed; repairing owner permissions and retrying: $output_base"
    if [[ -d "$output_base" ]]; then
        chmod u+rwx -- "$output_base" 2>/dev/null || true
        find "$output_base" -type d -exec chmod u+rwx {} + 2>/dev/null || true
    fi

    if ! rm -rf -- "$output_base" 2>/dev/null; then
        warn "Unable to remove Bazel cache; continuing teardown: $output_base"
    fi
    return 0
}

# Common pre-flight for both Next.js dev servers: clear stale lockfile and port
# squatters. Returns 1 if the port belongs to a process this worktree cannot own.
_prepare_next_port() {
    local label="$1"
    local web_dir="$2"
    local web_port="$3"
    local stale_lock=false

    local lock_file="$web_dir/.next/dev/lock"
    if [[ -f "$lock_file" ]]; then
        warn "检测到残留的 ${label}锁文件，清理中..."
        # The lock path is scoped to this worktree/frontend. Kill only its
        # holder and this frontend's port owner; a global `pkill next dev`
        # would terminate Admin or another worktree's devserver.
        lsof -t -- "$lock_file" 2>/dev/null | xargs kill 2>/dev/null || true
        lsof -ti :"$web_port" 2>/dev/null | xargs kill -9 2>/dev/null || true
        sleep 1
        rm -f "$lock_file"
        rm -rf "$web_dir/.next/cache"
        success "${label}锁文件和缓存已清理"
        stale_lock=true
    fi

    if [[ "$stale_lock" == false ]] && lsof -i :"$web_port" &>/dev/null; then
        warn "端口 $web_port 已被占用，跳过${label}启动"
        return 1
    fi
    return 0
}

# Launch Web through Bazel so the root npm_link_package dependencies remain
# visible to Next.js.
start_frontend() {
    source "$ENV_FILE"
    local web_dir="$SCRIPT_DIR/../../clients/web"
    local web_port="${WEB_PORT:-3000}"

    _prepare_next_port "前端" "$web_dir" "$web_port" || return 0

    if ! command -v bazel &>/dev/null; then
        error "未找到 bazel"
        return 1
    fi
    if ! command -v pnpm &>/dev/null; then
        error "未找到 pnpm，请先安装: npm install -g pnpm"
        return 1
    fi

    _install_root_deps_if_needed "前端依赖" "$web_dir/.next/cache" || return 1

    local log_file="$SCRIPT_DIR/web.log"
    local root_dir="$SCRIPT_DIR/../.."
    local bazel_output_base
    bazel_output_base=$(_frontend_bazel_output_base web)
    info "启动前端服务 (端口: $web_port, Bazel devserver)..."
    local saved_dir="$PWD"
    cd "$root_dir"
    # API_PROXY_TARGET drives next.config.ts rewrites through the worktree's
    # Traefik entrypoint. NEXT_PUBLIC_E2E enables only test-owned UI surfaces.
    API_PROXY_TARGET="http://localhost:$HTTP_PORT" \
    NEXT_PUBLIC_E2E="true" \
        bazel --output_base="$bazel_output_base" run //clients/web:next_dev -- --port "$web_port" > "$log_file" 2>&1 < /dev/null &
    disown $!
    cd "$saved_dir"

    local max_wait=60
    for ((i=1; i<=max_wait; i++)); do
        if curl -s "http://localhost:$web_port" &>/dev/null; then
            success "前端服务已启动 (http://localhost:$web_port)"
            return 0
        fi
        sleep 1
    done

    warn "前端服务启动中，请稍后访问 http://localhost:$web_port"
    echo "  查看日志: tail -f $log_file"
}

start_admin_frontend() {
    source "$ENV_FILE"
    local web_admin_dir="$SCRIPT_DIR/../../clients/web-admin"
    local web_admin_port="${WEB_ADMIN_PORT:-3001}"

    _prepare_next_port "Admin Console" "$web_admin_dir" "$web_admin_port" || return 0

    if ! command -v pnpm &>/dev/null; then
        error "未找到 pnpm，跳过 Admin Console 启动"
        return 0
    fi

    _install_root_deps_if_needed "Admin Console 依赖" "$web_admin_dir/.next/cache" || return 0

    local log_file="$SCRIPT_DIR/web-admin.log"
    local root_dir="$SCRIPT_DIR/../.."
    local bazel_output_base
    bazel_output_base=$(_frontend_bazel_output_base web-admin)
    info "启动 Admin Console (端口: $web_admin_port, Bazel devserver)..."
    local saved_dir="$PWD"
    cd "$root_dir"
    # web-admin derives its backend rewrite from PRIMARY_DOMAIN.
    PRIMARY_DOMAIN="localhost:$HTTP_PORT" \
        bazel --output_base="$bazel_output_base" run //clients/web-admin:next_dev -- --port "$web_admin_port" > "$log_file" 2>&1 < /dev/null &
    disown $!
    cd "$saved_dir"

    local max_wait=60
    for ((i=1; i<=max_wait; i++)); do
        if curl -s "http://localhost:$web_admin_port" &>/dev/null; then
            success "Admin Console 已启动 (http://localhost:$web_admin_port)"
            return 0
        fi
        sleep 1
    done

    warn "Admin Console 启动中，请稍后访问 http://localhost:$web_admin_port"
    echo "  查看日志: tail -f $log_file"
}
