#!/usr/bin/env bash
# shellcheck disable=SC1091,SC2016,SC2034,SC2329
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
DEV_DIR="$ROOT_DIR/deploy/dev"
FRONTEND_SERVICES="$DEV_DIR/lib/frontend_services.sh"
LIFECYCLE="$DEV_DIR/lib/lifecycle.sh"
DEV_ENTRYPOINT="$DEV_DIR/dev.sh"

fail() {
    echo "frontend services contract failed: $*" >&2
    exit 1
}

bash -n "$DEV_ENTRYPOINT" "$FRONTEND_SERVICES" "$LIFECYCLE"

worktree_line=$(grep -nF 'source "$SCRIPT_DIR/lib/worktree.sh"' "$DEV_ENTRYPOINT" | cut -d: -f1)
frontend_line=$(grep -nF 'source "$SCRIPT_DIR/lib/frontend_services.sh"' "$DEV_ENTRYPOINT" | cut -d: -f1)
lifecycle_line=$(grep -nF 'source "$SCRIPT_DIR/lib/lifecycle.sh"' "$DEV_ENTRYPOINT" | cut -d: -f1)
[[ -n "$worktree_line" && -n "$frontend_line" && -n "$lifecycle_line" ]] \
    || fail "dev.sh is missing a lifecycle source"
(( worktree_line < frontend_line && frontend_line < lifecycle_line )) \
    || fail "source order must be worktree -> frontend_services -> lifecycle"

SCRIPT_DIR="$DEV_DIR"
get_worktree_name() { echo "coverage-contract-worktree"; }
# shellcheck source=lib/frontend_services.sh
source "$FRONTEND_SERVICES"
# shellcheck source=lib/lifecycle.sh
source "$LIFECYCLE"

tmp_root=$(mktemp -d)
trap 'rm -rf "$tmp_root"' EXIT
fake_root="$tmp_root/workspace"
fake_dev="$fake_root/deploy/dev"
fake_runtime="$tmp_root/runtime"
mkdir -p "$fake_dev" "$fake_runtime"

test_dependency_fingerprint_lifecycle() (
    SCRIPT_DIR="$fake_dev"
    local lockfile="$fake_root/pnpm-lock.yaml"
    local node_modules="$fake_root/node_modules"
    local cache_dir="$fake_root/clients/web/.next/cache"
    local install_log="$tmp_root/pnpm-install.log"
    local current_hash
    mkdir -p "$node_modules" "$cache_dir"
    echo "lock-version: 1" > "$lockfile"
    current_hash=$(md5 -q "$lockfile" 2>/dev/null || md5sum "$lockfile" | cut -d' ' -f1)
    echo "$current_hash" > "$node_modules/.pnpm-lock-hash"

    info() { :; }
    success() { :; }
    error() { printf 'error:%s\n' "$*" >> "$install_log"; }
    pnpm() { printf 'pnpm:%s pwd:%s\n' "$*" "$PWD" >> "$install_log"; }

    _install_root_deps_if_needed "前端依赖" "$cache_dir"
    [[ ! -e "$install_log" ]] || fail "fresh dependency fingerprint unexpectedly ran pnpm"
    [[ -d "$cache_dir" ]] || fail "fresh dependency fingerprint removed the cache"

    echo "stale" > "$node_modules/.pnpm-lock-hash"
    _install_root_deps_if_needed "前端依赖" "$cache_dir"
    grep -Fq "pnpm:install --frozen-lockfile pwd:$fake_root" "$install_log" \
        || fail "stale dependency fingerprint did not run pnpm in the workspace root"
    [[ $(cat "$node_modules/.pnpm-lock-hash") == "$current_hash" ]] \
        || fail "successful install did not publish the current lockfile fingerprint"
    [[ ! -e "$cache_dir" ]] || fail "successful reinstall did not clear the stale Next cache"

    mkdir -p "$cache_dir"
    echo "still-stale" > "$node_modules/.pnpm-lock-hash"
    pnpm() { return 1; }
    if _install_root_deps_if_needed "前端依赖" "$cache_dir"; then
        fail "failed pnpm install was reported as successful"
    fi
    [[ $(cat "$node_modules/.pnpm-lock-hash") == "still-stale" ]] \
        || fail "failed install replaced the dependency fingerprint"
    [[ -d "$cache_dir" ]] || fail "failed install removed the existing cache"
)

test_port_ownership_lifecycle() (
    SCRIPT_DIR="$fake_dev"
    local web_dir="$fake_root/clients/web"
    local lock_file="$web_dir/.next/dev/lock"
    local cache_dir="$web_dir/.next/cache"
    local command_log="$tmp_root/port-commands.log"
    mkdir -p "$cache_dir"
    : > "$lock_file"

    warn() { :; }
    success() { :; }
    sleep() { :; }
    lsof() {
        printf 'lsof:%s\n' "$*" >> "$command_log"
        case "$*" in
            "-t -- $lock_file") echo 101 ;;
            "-ti :34100") echo 202 ;;
            *) return 1 ;;
        esac
    }
    xargs() {
        local input
        input=$(cat)
        printf 'xargs:%s input:%s\n' "$*" "$input" >> "$command_log"
    }

    _prepare_next_port "前端" "$web_dir" 34100
    [[ ! -e "$lock_file" && ! -e "$cache_dir" ]] \
        || fail "stale frontend lock cleanup left its lock or cache behind"
    grep -Fq "lsof:-t -- $lock_file" "$command_log" \
        || fail "stale cleanup did not inspect the lock owner"
    grep -Fq "lsof:-ti :34100" "$command_log" \
        || fail "stale cleanup did not inspect the frontend port owner"
    grep -Fq "xargs:kill input:101" "$command_log" \
        || fail "stale cleanup did not terminate the lock owner"
    grep -Fq "xargs:kill -9 input:202" "$command_log" \
        || fail "stale cleanup did not terminate the port owner"

    lsof() { [[ "$*" == "-i :34100" ]]; }
    if _prepare_next_port "前端" "$web_dir" 34100; then
        fail "foreign port owner was treated as available"
    fi
    lsof() { return 1; }
    _prepare_next_port "前端" "$web_dir" 34100 \
        || fail "free frontend port was rejected"
)

test_missing_frontend_tools() (
    SCRIPT_DIR="$fake_dev"
    ENV_FILE="$tmp_root/frontend-tools.env"
    cat > "$ENV_FILE" <<'EOF'
WEB_PORT=34100
HTTP_PORT=34102
EOF
    local errors="$tmp_root/frontend-tool-errors.log"
    _prepare_next_port() { return 0; }
    error() { printf '%s\n' "$*" >> "$errors"; }
    command() {
        if [[ "$1" == "-v" && "$2" == "bazel" ]]; then return 1; fi
        builtin command "$@"
    }

    if start_frontend; then
        fail "Web startup succeeded without Bazel"
    fi
    grep -Fq "未找到 bazel" "$errors" || fail "missing Bazel error was not reported"

    : > "$errors"
    command() {
        if [[ "$1" == "-v" && "$2" == "pnpm" ]]; then return 1; fi
        return 0
    }
    if start_frontend; then
        fail "Web startup succeeded without pnpm"
    fi
    grep -Fq "未找到 pnpm" "$errors" || fail "missing pnpm error was not reported"
)

SCRIPT_DIR="$fake_dev"
ENV_FILE="$tmp_root/frontend.env"
cat > "$ENV_FILE" <<'EOF'
WEB_PORT=34100
WEB_ADMIN_PORT=34101
HTTP_PORT=34102
EOF
AGENTSMESH_DEV_BAZEL_ROOT="$tmp_root/bazel"
CALL_LOG="$tmp_root/calls.log"
: > "$CALL_LOG"

info() { :; }
success() { :; }
warn() { :; }
error() { fail "$*"; }
pnpm() { :; }
curl() { return 0; }
lsof() { return 1; }
stop_host_services() { :; }
_runtime_dir() { echo "$fake_runtime"; }
_prepare_next_port() { return 0; }
_install_root_deps_if_needed() { return 0; }
bazel() {
    printf 'api=%s e2e=%s domain=%s args=%s\n' \
        "${API_PROXY_TARGET:-}" \
        "${NEXT_PUBLIC_E2E:-}" \
        "${PRIMARY_DOMAIN:-}" \
        "$*" >> "$CALL_LOG"
}

web_base=$(_frontend_bazel_output_base web)
admin_base=$(_frontend_bazel_output_base web-admin)
[[ "$web_base" == "$tmp_root/bazel/coverage-contract-worktree-bazel-web" ]] \
    || fail "unexpected Web output base: $web_base"
[[ "$admin_base" == "$tmp_root/bazel/coverage-contract-worktree-bazel-web-admin" ]] \
    || fail "unexpected Admin output base: $admin_base"
[[ "$web_base" != "$admin_base" ]] || fail "frontends must not share a Bazel output base"

start_frontend
start_admin_frontend
for _ in {1..100}; do
    [[ $(wc -l < "$CALL_LOG" 2>/dev/null || echo 0) -ge 2 ]] && break
    sleep 0.01
done

grep -Fq "api=http://localhost:34102 e2e=true domain= args=--output_base=$web_base run //clients/web:next_dev -- --port 34100" "$CALL_LOG" \
    || fail "Web launch did not use its isolated output base and proxy environment"
grep -Fq "api= e2e= domain=localhost:34102 args=--output_base=$admin_base run //clients/web-admin:next_dev -- --port 34101" "$CALL_LOG" \
    || fail "Admin launch did not use its isolated output base and proxy environment"

mkdir -p "$web_base" "$admin_base"
ENV_FILE="$tmp_root/not-initialized.env"
clean
[[ ! -e "$web_base" && ! -e "$admin_base" ]] \
    || fail "clean did not remove both isolated output bases"
grep -Fq "args=--output_base=$web_base shutdown" "$CALL_LOG" \
    || fail "clean did not shut down the Web Bazel server"
grep -Fq "args=--output_base=$admin_base shutdown" "$CALL_LOG" \
    || fail "clean did not shut down the Admin Bazel server"

test_resilient_bazel_cache_teardown() (
    SCRIPT_DIR="$fake_dev"
    ENV_FILE="$tmp_root/clean.env"
    AGENTSMESH_DEV_BAZEL_ROOT="$tmp_root/clean-bazel"
    local clean_runtime="$tmp_root/clean-runtime"
    local clean_log="$tmp_root/clean.log"
    local sentinel="$tmp_root/must-survive"
    local resilient_web_base resilient_admin_base
    local web_rm_attempts=0
    resilient_web_base=$(_frontend_bazel_output_base web)
    resilient_admin_base=$(_frontend_bazel_output_base web-admin)

    cat > "$ENV_FILE" <<'EOF'
WEB_PORT=34200
WEB_ADMIN_PORT=34201
COMPOSE_PROJECT_NAME=coverage-clean
EOF
    mkdir -p \
        "$resilient_web_base/readonly/nested" \
        "$resilient_admin_base" \
        "$clean_runtime"
    : > "$resilient_web_base/readonly/nested/cache-entry"
    : > "$clean_runtime/runtime-entry"
    : > "$sentinel"
    chmod 500 "$resilient_web_base/readonly/nested" "$resilient_web_base/readonly"
    : > "$clean_log"

    info() { :; }
    success() { :; }
    warn() { printf 'warn:%s\n' "$*" >> "$clean_log"; }
    chmod() {
        printf 'chmod:%s\n' "$*" >> "$clean_log"
        command chmod "$@"
    }
    lsof() { return 1; }
    stop_host_services() { printf 'stop-host-services\n' >> "$clean_log"; }
    _runtime_dir() { echo "$clean_runtime"; }
    bazel() { printf 'bazel:%s\n' "$*" >> "$clean_log"; }
    docker() { printf 'docker:%s\n' "$*" >> "$clean_log"; }
    rm() {
        if [[ $# -eq 3 && "$1" == "-rf" && "$2" == "--" ]]; then
            if [[ "$3" == "$resilient_web_base" ]]; then
                web_rm_attempts=$((web_rm_attempts + 1))
                printf 'web-rm-attempt:%s\n' "$web_rm_attempts" >> "$clean_log"
                if [[ "$web_rm_attempts" -eq 1 ]]; then
                    return 1
                fi
            elif [[ "$3" == "$resilient_admin_base" ]]; then
                printf 'forced-rm-failure:%s\n' "$3" >> "$clean_log"
                return 1
            fi
        fi
        command rm "$@"
    }

    clean

    [[ ! -e "$resilient_web_base" ]] \
        || fail "clean did not repair and remove the read-only Web output base"
    [[ -d "$resilient_admin_base" ]] \
        || fail "forced persistent cache deletion failure was not exercised"
    [[ ! -e "$clean_runtime" ]] \
        || fail "cache deletion failure interrupted runtime cleanup"
    [[ ! -e "$ENV_FILE" ]] \
        || fail "cache deletion failure interrupted environment cleanup"
    [[ -e "$sentinel" ]] \
        || fail "cache permission repair escaped its output base"
    grep -Fq "repairing owner permissions and retrying: $resilient_web_base" "$clean_log" \
        || fail "read-only cache did not exercise permission repair"
    grep -Fq "chmod:u+rwx -- $resilient_web_base" "$clean_log" \
        || fail "cache retry did not repair owner directory permissions"
    [[ $(grep -Fc "web-rm-attempt:" "$clean_log") -eq 2 ]] \
        || fail "Web output base deletion was not retried exactly once"
    grep -Fq "bazel:--output_base=$resilient_admin_base shutdown" "$clean_log" \
        || fail "Web cache retry interrupted Admin Bazel shutdown"
    grep -Fq "Unable to remove Bazel cache; continuing teardown: $resilient_admin_base" "$clean_log" \
        || fail "persistent cache deletion failure did not emit a warning"
    grep -Fq "docker:compose down -v --remove-orphans" "$clean_log" \
        || fail "cache deletion failure interrupted Docker teardown"

    command rm -rf "$resilient_admin_base"
)

test_resilient_bazel_cache_teardown

if grep -Eq 'pkill[[:space:]]+-f[[:space:]]+"?next dev' "$FRONTEND_SERVICES"; then
    fail "frontend cleanup must not kill every worktree's Next.js process"
fi

echo "frontend services contract: PASS"
