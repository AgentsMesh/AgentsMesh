# shellcheck shell=bash
# host_services.sh — host-side ibazel service lifecycle.
#
# Backend / relay run on the developer host (post air → ibazel migration);
# this module owns their build / launch / teardown / health checks.
# Runner stays in docker but its binary is also bazel-built — see
# `build_runner_binary` for the cross-compile path that feeds the docker
# image's COPY.

# Wait for an HTTP endpoint to return success. 1-second polling, default
# 40 attempts (= 40s max). Used for backend / relay health checks.
_wait_http() {
    local url="$1" name="$2" max="${3:-40}"
    for ((i=1; i<=max; i++)); do
        if curl -sf "$url" >/dev/null 2>&1; then return 0; fi
        sleep 1
    done
    error "$name 健康检查超时 ($url)"
    return 1
}

# Background-launch a service via ibazel. Args: name target [extra args...].
# Writes pid + log under runtime/<name>/. Reaps an orphan from a prior run
# so re-runs are idempotent.
_launch_ibazel() {
    local name="$1" target="$2"; shift 2
    local rt_dir
    rt_dir="$(_runtime_dir)/$name"
    mkdir -p "$rt_dir"
    local pid_file="$rt_dir/$name.pid"
    local log_file="$rt_dir/$name.log"

    if [[ -f "$pid_file" ]]; then
        local old
        old=$(cat "$pid_file")
        if kill -0 "$old" 2>/dev/null; then
            kill -TERM "$old" 2>/dev/null || true
            sleep 1
        fi
        rm -f "$pid_file"
    fi

    info "启动 host service: $name (target: $target)"
    local repo_root="$SCRIPT_DIR/../.."
    (
        cd "$repo_root"
        nohup ibazel run "$target" "$@" > "$log_file" 2>&1 &
        echo $! > "$pid_file"
        disown
    )
}

# Cross-compile the runner binary for linux/amd64 and copy it to
# deploy/dev/runner-binary so docker compose can COPY it into the runner
# image. macOS bazel-bin is a symlink chain to /private/var/... that
# docker build can't follow across, hence `cp -L`.
build_runner_binary() {
    info "Bazel build runner binary (linux/amd64)..."
    local repo_root="$SCRIPT_DIR/../.."
    (
        cd "$repo_root"
        bazel build //runner/cmd/runner:runner \
            --platforms=@rules_go//go/toolchain:linux_amd64
    ) || {
        error "bazel build runner 失败"
        return 1
    }
    rm -f "$SCRIPT_DIR/runner-binary"
    cp -L "$repo_root/bazel-bin/runner/cmd/runner/runner_/runner" \
        "$SCRIPT_DIR/runner-binary"
    chmod +x "$SCRIPT_DIR/runner-binary"
    success "Runner binary 已编译并复制到 build context"
}

# Cross-compile the e2e-mock-agent (sibling to build_runner_binary) so the
# Dockerfile can COPY it into the runner image. The mock agent is the
# executable behind the `e2e-echo` AgentFile (PTY+ACP modes), used by
# mcp-e2e / envbundle-e2e / acp-ui-e2e — i.e. the runtime under test in
# every harness that creates a pod without a real LLM CLI.
build_mock_agent_binary() {
    info "Bazel build e2e-mock-agent binary (linux/amd64)..."
    local repo_root="$SCRIPT_DIR/../.."
    (
        cd "$repo_root"
        bazel build //runner/internal/agents/mockagent/cmd/e2e-mock-agent:e2e-mock-agent \
            --platforms=@rules_go//go/toolchain:linux_amd64
    ) || {
        error "bazel build e2e-mock-agent 失败"
        return 1
    }
    rm -f "$SCRIPT_DIR/e2e-mock-agent-binary"
    cp -L "$repo_root/bazel-bin/runner/internal/agents/mockagent/cmd/e2e-mock-agent/e2e-mock-agent_/e2e-mock-agent" \
        "$SCRIPT_DIR/e2e-mock-agent-binary"
    chmod +x "$SCRIPT_DIR/e2e-mock-agent-binary"
    success "e2e-mock-agent binary 已编译并复制到 build context"
}

# Pre-build the binary (no health budget pressure), then ibazel run for
# the actual launch — the launcher's `bazel run` reuses the cached binary
# so `_wait_http` only waits on real startup time, not Bazel compile.
# Critical on cold-cache CI where compile alone is 5+ min.
start_backend_host() {
    source "$ENV_FILE"
    local repo_root="$SCRIPT_DIR/../.."
    mkdir -p "$repo_root/backend/logs"

    # Mirrors what the docker compose backend service used to set, but
    # DB / redis / minio talk through host port forwards.
    export DB_HOST=localhost
    export DB_PORT="$POSTGRES_PORT"
    export DB_USER=agentsmesh
    export DB_PASSWORD="${POSTGRES_PASSWORD:-agentsmesh_dev}"
    export DB_NAME=agentsmesh
    export DB_SSLMODE=disable
    export REDIS_URL="redis://localhost:${REDIS_PORT}"
    export JWT_SECRET="${JWT_SECRET:-dev-jwt-secret-change-in-production}"
    export INTERNAL_API_SECRET="${INTERNAL_API_SECRET:-dev-internal-secret}"
    export SERVER_ADDRESS=":${BACKEND_HTTP_PORT}"
    export GRPC_ADDRESS=":${BACKEND_GRPC_PORT}"
    export GRPC_PUBLIC_ENDPOINT="grpcs://127.0.0.1:${GRPC_PORT}"
    export DEBUG=true
    # Disable IP/User rate limiters in dev + CI. Without this, e2e suites on
    # fast hardware (self-hosted runners, local ramped-up dev loops) overflow
    # the 20 req/min `/auth/*` cap while authenticating across specs and
    # surface as 429s — see backend/internal/middleware/ratelimit.go.
    export RATE_LIMIT_DISABLED=true
    export PRIMARY_DOMAIN="${PRIMARY_DOMAIN}"
    export USE_HTTPS="${USE_HTTPS:-false}"
    # Webhook allowlist for trigger-fire e2e: needs `localhost` so the
    # test's local HTTP listener is reachable. Deliberately excludes
    # bare `127.0.0.1` — security-guards e2e verifies that loopback IPs
    # are rejected, and listing the IP literal would short-circuit that
    # check (allowlist is exact-match before the SSRF policy fires).
    export BLOCKSTORE_WEBHOOK_ALLOW_HOSTS="host.docker.internal,host.lan,localhost"
    export CORS_ALLOWED_ORIGINS="http://localhost:${HTTP_PORT},http://127.0.0.1:${HTTP_PORT},http://localhost:${WEB_PORT},http://127.0.0.1:${WEB_PORT},http://localhost:${WEB_ADMIN_PORT},http://127.0.0.1:${WEB_ADMIN_PORT}"
    export LOG_LEVEL=debug
    export LOG_FORMAT=text
    export LOG_FILE="$repo_root/backend/logs/agentsmesh.log"
    export EMAIL_PROVIDER=console
    export STORAGE_ENDPOINT="localhost:${MINIO_API_PORT}"
    export STORAGE_PUBLIC_ENDPOINT="localhost:${MINIO_API_PORT}"
    export STORAGE_REGION=us-east-1
    export STORAGE_BUCKET=agentsmesh
    export STORAGE_ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
    export STORAGE_SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"
    export STORAGE_USE_SSL=false
    export STORAGE_USE_PATH_STYLE=true
    export STORAGE_MAX_FILE_SIZE=10
    export STORAGE_ALLOWED_TYPES="image/jpeg,image/png,image/gif,image/webp,application/pdf"
    export DEPLOYMENT_TYPE="${DEPLOYMENT_TYPE:-global}"
    export PAYMENT_MOCK="${PAYMENT_MOCK:-false}"
    export PKI_CA_CERT_FILE="$SCRIPT_DIR/ssl/ca.crt"
    export PKI_CA_KEY_FILE="$SCRIPT_DIR/ssl/ca.key"
    export PKI_VALIDITY_DAYS=365
    export GEO_MMDB_PATH="${GEO_MMDB_PATH:-}"
    export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:${OTEL_GRPC_PORT}"
    export OTEL_SERVICE_NAME=agentsmesh-backend
    export OTEL_TRACES_SAMPLER_ARG=1.0

    info "构建 backend 二进制 (bazel build)..."
    # Include the go_proto_library so protoc + protoc-gen-go-grpc C++
    # tool-chain compilation lands in the disk cache before ibazel takes
    # over. Without this, ibazel's first `bazel run` starts the protoc
    # build (5+ min on cold CI) while the health check is already
    # ticking, causing 90s timeouts.
    (cd "$repo_root" && bazel build //backend/cmd/server:server //proto/runner/v1:runner_go_proto) || {
        error "Backend 构建失败"
        return 1
    }

    _launch_ibazel backend //backend/cmd/server:server

    # 480s budget covers cold-CI's first protoc + protoc-gen-go-grpc C++
    # toolchain compile (~6 min, 1252 actions). Once GHA's bazel disk
    # cache warms up after one successful run, subsequent invocations
    # land in <10 s. Pre-build (above) does *not* warm this for ibazel
    # because rules_go's `bazel run` invocation walks an analysis path
    # whose action keys differ from `bazel build :runner_go_proto`'s.
    if ! _wait_http "http://localhost:${BACKEND_HTTP_PORT}/health" backend 480; then
        error "Backend 启动失败，查看日志: $(_runtime_dir)/backend/backend.log"
        echo "--- backend.log (last 80 lines) ---" >&2
        tail -80 "$(_runtime_dir)/backend/backend.log" >&2 || true
        return 1
    fi
    success "Backend 已就绪 (host :${BACKEND_HTTP_PORT}, gRPC :${BACKEND_GRPC_PORT})"
}

# Relay reads SERVER_PORT (not SERVER_ADDRESS like backend) — see
# relay/internal/config/config.go.
start_relay_host() {
    source "$ENV_FILE"
    local repo_root="$SCRIPT_DIR/../.."
    export SERVER_HOST="0.0.0.0"
    export SERVER_PORT="${RELAY_HTTP_PORT}"
    export WS_READ_BUFFER_SIZE=4096
    export WS_WRITE_BUFFER_SIZE=4096
    export JWT_SECRET="${JWT_SECRET:-dev-jwt-secret-change-in-production}"
    export BACKEND_URL="http://localhost:${HTTP_PORT}"
    export INTERNAL_API_SECRET="${INTERNAL_API_SECRET:-dev-internal-secret}"
    export RELAY_ID=dev-relay-1
    export RELAY_REGION=local
    export RELAY_CAPACITY=1000
    export PRIMARY_DOMAIN="${PRIMARY_DOMAIN}"
    export USE_HTTPS="${USE_HTTPS:-false}"
    export SESSION_KEEP_ALIVE_DURATION=30s
    export DEBUG=true
    export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:${OTEL_GRPC_PORT}"
    export OTEL_SERVICE_NAME=agentsmesh-relay
    export OTEL_TRACES_SAMPLER_ARG=1.0

    info "构建 relay 二进制 (bazel build)..."
    (cd "$repo_root" && bazel build //relay/cmd/relay:relay) || {
        error "Relay 构建失败"
        return 1
    }

    _launch_ibazel relay //relay/cmd/relay:relay

    if ! _wait_http "http://localhost:${RELAY_HTTP_PORT}/health" relay 60; then
        error "Relay 启动失败，查看日志: $(_runtime_dir)/relay/relay.log"
        echo "--- relay.log (last 80 lines) ---" >&2
        tail -80 "$(_runtime_dir)/relay/relay.log" >&2 || true
        return 1
    fi
    success "Relay 已就绪 (host :${RELAY_HTTP_PORT})"
}

# Stop host backend / relay. Runner stays in docker, so `clean` handles
# it via `docker compose down`.
stop_host_services() {
    local rt_root
    rt_root="$(_runtime_dir)"
    [[ -d "$rt_root" ]] || return 0
    for svc in backend relay; do
        local pid_file="$rt_root/$svc/$svc.pid"
        if [[ -f "$pid_file" ]]; then
            local pid
            pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                info "停止 host $svc (pid: $pid)..."
                # ibazel spawns a child process; group-kill catches both.
                kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
                pkill -P "$pid" 2>/dev/null || true
            fi
            rm -f "$pid_file"
        fi
    done
    pkill -f 'ibazel run //backend/cmd/server' 2>/dev/null || true
    pkill -f 'ibazel run //relay/cmd/relay' 2>/dev/null || true
}
