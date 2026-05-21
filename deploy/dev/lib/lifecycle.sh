# shellcheck shell=bash
# lifecycle.sh — start / stop / status / banner.
#
# Frontend launch (web + admin) goes through Bazel's `next_dev` so the
# `@agentsmesh/*` internal packages — linked at //:node_modules via Bazel
# `npm_link_package`, not by pnpm — are visible to Next.js.
#
# `clean` tears down everything dev.sh created (host pids, frontend ports,
# docker volumes, .env). `reset_runners` is the targeted "rebuild + restart
# the runner container" path used after a runner-only code change.

# Banner / usage / docker-compose-up are factored out of the original main()
# so the entry point is just orchestration.

print_banner() {
    echo ""
    echo "=========================================="
    echo "  AgentsMesh 开发环境初始化"
    echo "=========================================="
    echo ""
}

print_usage() {
    cat << 'EOF'
用法:
  bazel run //deploy/dev:up                 # 一键启动完整开发环境
  bazel run //deploy/dev:backend_only       # 仅启动 docker + host backend/relay (CI)
  bazel run //deploy/dev:rebuild_runner     # 重 build runner binary + 重启容器
  bazel run //deploy/dev:reset_runners      # 重启 host runner+relay (backend 不动)
  bazel run //deploy/dev:clean              # 停止并清理所有服务

  或直接调脚本（backward-compat）:
  ./dev.sh [--backend-only|--rebuild-runner|--reset-runners|--clean|--help]

  改动 backend / relay 源码: ibazel 自动重 build (host)
  改动 runner 源码:        bazel run //deploy/dev:rebuild_runner

前端日志: tail -f deploy/dev/web.log
EOF
}

# `docker compose up -d --build` with a 3-attempt retry loop. The build
# context is small but the npm registry / Docker Hub fetch is flaky on
# fresh CI runners, so retries beat hard-fail every time.
docker_compose_up() {
    info "启动 Docker 基础设施 + runner (首次可能需要几分钟)..."
    local up_attempt=0
    local up_max=3
    while [ $up_attempt -lt $up_max ]; do
        up_attempt=$((up_attempt + 1))
        # set -o pipefail so docker compose's non-zero exit (auth.docker.io
        # token timeouts, build failures) actually fails the pipe — without
        # it grep returns 0 even if compose crashed and the loop exits
        # success'fully' while postgres is missing.
        if (set -o pipefail; docker compose up -d --build --quiet-pull 2>&1 | grep -v "^#" | grep -v "^\[" | grep -v "^$"); then
            break
        fi
        if [ $up_attempt -eq $up_max ]; then
            error "Docker compose up failed after $up_max attempts"
            exit 1
        fi
        warn "compose up failed (attempt $up_attempt/$up_max) — retrying in 10s"
        sleep 10
    done
    success "Docker 基础设施已启动"
}

wait_for_postgres() {
    local pg_container="${COMPOSE_PROJECT_NAME}-postgres-1"
    info "等待 PostgreSQL 就绪..."
    if ! wait_for_service "$pg_container" "pg_isready -U agentsmesh"; then
        error "PostgreSQL 启动超时"
        exit 1
    fi
    success "PostgreSQL 已就绪"
}

# Kill stale runner CLI processes (in case anyone installed agentsmesh-runner
# from `cargo install` or similar), rebuild the binary, then `docker compose
# up -d --build runner` to pick up the fresh binary via the runner-binary
# COPY in runner.Dockerfile.
reset_runners() {
    if [[ -f "$ENV_FILE" ]]; then
        source "$ENV_FILE"
    fi

    echo ""
    echo "=========================================="
    echo "  Reset Runner (rebuild bazel binary + restart container)"
    echo "=========================================="
    echo ""

    if pgrep -f "agentsmesh-runner" &>/dev/null; then
        info "停止本地 agentsmesh-runner 进程..."
        pkill -f "agentsmesh-runner" 2>/dev/null || true
        sleep 1
        pkill -9 -f "agentsmesh-runner" 2>/dev/null || true
    fi

    build_runner_binary || return 1
    build_mock_agent_binary || return 1

    cd "$SCRIPT_DIR"
    info "重建并重启 runner 容器..."
    docker compose up -d --build runner 2>&1 | grep -v "^#" | grep -v warning || true
    success "Runner 容器已重启 (binary 来自 bazel build)"

    echo ""
}

# Tear down everything dev.sh creates: host service pids, frontend port
# squatters, docker volumes, .env. Safe to re-run.
clean() {
    if [[ -f "$ENV_FILE" ]]; then
        source "$ENV_FILE"
    fi
    local web_port="${WEB_PORT:-3000}"
    local web_admin_port="${WEB_ADMIN_PORT:-3001}"

    info "停止 host-side ibazel 服务..."
    stop_host_services
    success "host-side 服务已停止"

    if lsof -i :"$web_port" &>/dev/null; then
        info "停止前端服务 (端口: $web_port)..."
        lsof -ti :"$web_port" | xargs kill -9 2>/dev/null || true
        success "前端服务已停止"
    fi

    if lsof -i :"$web_admin_port" &>/dev/null; then
        info "停止 Admin Console (端口: $web_admin_port)..."
        lsof -ti :"$web_admin_port" | xargs kill -9 2>/dev/null || true
        success "Admin Console 已停止"
    fi

    rm -f "$SCRIPT_DIR/web.log"
    rm -f "$SCRIPT_DIR/web-admin.log"
    rm -rf "$(_runtime_dir)"

    if [[ -f "$ENV_FILE" ]]; then
        info "清理 Docker 环境: ${COMPOSE_PROJECT_NAME:-agentsmesh}..."
        cd "$SCRIPT_DIR"
        docker compose down -v --remove-orphans 2>/dev/null || true
        rm -f "$ENV_FILE"
        success "清理完成"
    else
        warn "Docker 环境未初始化"
    fi
}

show_result() {
    source "$ENV_FILE"

    echo ""
    echo "=========================================="
    echo "  AgentsMesh 开发环境已就绪!"
    echo "=========================================="
    echo ""
    echo "  前端:       http://localhost:$WEB_PORT"
    echo "  Admin:      http://localhost:$WEB_ADMIN_PORT"
    echo "  API:        http://localhost:$HTTP_PORT/api  (→ host backend :$BACKEND_HTTP_PORT)"
    echo "  Relay:      ws://localhost:$HTTP_PORT/relay  (→ host relay :$RELAY_HTTP_PORT)"
    echo "  gRPC mTLS:  grpcs://localhost:$GRPC_PORT      (→ host backend :$BACKEND_GRPC_PORT)"
    echo ""
    echo "  Host services (ibazel hot-reload):"
    echo "    backend  日志: tail -f deploy/dev/runtime/backend/backend.log"
    echo "    relay    日志: tail -f deploy/dev/runtime/relay/relay.log"
    echo ""
    echo "  Docker runner (bazel-built binary, no hot reload):"
    echo "    日志: docker compose logs -f runner"
    echo "    重 build: ./dev.sh --rebuild-runner"
    echo ""
    echo "  测试账号:   dev@agentsmesh.local / devpass123"
    echo "  管理员:     admin@agentsmesh.local / adminpass123"
    echo ""
    echo "  其他服务:"
    echo "    Gitea:    http://localhost:$GITEA_HTTP_PORT (gitea-admin / gitea-admin-123)"
    echo "    Traefik:  http://localhost:$TRAEFIK_DASHBOARD_PORT (Dashboard)"
    echo "    Adminer:  http://localhost:$ADMINER_PORT"
    echo "    MinIO:    http://localhost:$MINIO_CONSOLE_PORT"
    echo "    Jaeger:   http://localhost:$JAEGER_UI_PORT (Tracing UI)"
    echo ""
    echo "  停止: ./dev.sh --clean"
    echo "  仅重 build runner: ./dev.sh --rebuild-runner"
    echo ""
}

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

# Common pre-flight for both Next.js dev servers: clear stale lockfile +
# port squatters. Returns 1 if the port is held by something we can't
# safely kick (i.e., not our own stale Next.js process).
_prepare_next_port() {
    local label="$1"      # "前端" / "Admin Console"
    local web_dir="$2"    # absolute path to clients/web or web-admin
    local web_port="$3"
    local stale_lock=false

    local lock_file="$web_dir/.next/dev/lock"
    if [[ -f "$lock_file" ]]; then
        warn "检测到残留的 ${label}锁文件，清理中..."
        # Only kill `next dev` process for the web frontend — admin keeps
        # using the lsof fallback because both frontends share the same
        # `next dev` process name and we don't want one cleanup to kill
        # the other.
        if [[ "$label" == "前端" ]]; then
            pkill -f "next dev" 2>/dev/null || true
        fi
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

# Launch the Next.js web frontend via Bazel's `next_dev` devserver. We
# can't use plain `next dev` from clients/web/ because the
# `@agentsmesh/*` internal packages are linked at the workspace root
# via Bazel `npm_link_package`, not by pnpm — so Next.js running outside
# Bazel's sandbox can't resolve them.
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
    info "启动前端服务 (端口: $web_port, Bazel devserver)..."
    local saved_dir="$PWD"
    cd "$root_dir"
    # API_PROXY_TARGET drives next.config.ts rewrites: /api/* → traefik
    # → host backend. Without it, /api/auth/login 404s and the UI can't
    # log in. HTTP_PORT is traefik's worktree-allocated entrypoint.
    API_PROXY_TARGET="http://localhost:$HTTP_PORT" \
        bazel run //clients/web:next_dev -- --port "$web_port" > "$log_file" 2>&1 < /dev/null &
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
    info "启动 Admin Console (端口: $web_admin_port, Bazel devserver)..."
    local saved_dir="$PWD"
    cd "$root_dir"
    # web-admin's next.config rewrites use PRIMARY_DOMAIN to compute the
    # backend URL (its fallback is the prod-only localhost:10000, which
    # never matches a worktree). Pin it to traefik so /api/* proxies.
    PRIMARY_DOMAIN="localhost:$HTTP_PORT" \
        bazel run //clients/web-admin:next_dev -- --port "$web_admin_port" > "$log_file" 2>&1 < /dev/null &
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
