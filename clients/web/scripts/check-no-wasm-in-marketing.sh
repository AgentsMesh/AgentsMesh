#!/usr/bin/env bash
# Verify that pre-dashboard chunks don't pull in wasm. Run after
# `bazel build //clients/web:next`. Exits non-zero if any leak.
#
# Architectural invariant (Plan I1 + auth-light rollout):
#   Marketing routes AND the entire (auth) route group must not load the
#   40MB wasm bundle — they go through @/lib/light-auth + light-session.
#   Only (dashboard), popout, and blocks-embed are allowed to boot wasm.

set -euo pipefail

NEXT_DIR="${NEXT_DIR:-bazel-bin/clients/web/.next}"
CHUNKS_DIR="${NEXT_DIR}/static/chunks"

if [[ ! -d "${CHUNKS_DIR}" ]]; then
  echo "FAIL: ${CHUNKS_DIR} not found. Run: bazel build //clients/web:next"
  exit 2
fi

# Pre-dashboard route chunks: marketing + (auth). Must not contain
# WasmProvider / initWasmCore / any wasm-bindgen class names.
LEAKS=$(
  find "${CHUNKS_DIR}/app" -maxdepth 4 -name "*.js" \
    -not -path "*\(dashboard\)*" \
    -not -path "*popout*" \
    -not -path "*blocks-embed*" \
    -print0 \
  | xargs -0 grep -l 'WasmProvider\|initWasmCore\|WasmApiClient\|WasmAuthManager\|wasm_pkg\|agentsmesh-wasm' 2>/dev/null \
  || true
)

if [[ -n "${LEAKS}" ]]; then
  echo "FAIL: pre-dashboard chunks contain wasm symbols:"
  echo "${LEAKS}" | sed 's/^/  /'
  exit 1
fi

echo "PASS: no wasm symbols in marketing or (auth) chunks"

# Confirm wasm-bound layouts DO contain WasmProvider (positive check —
# catches a future regression where someone deletes the (dashboard)
# WasmProvider import and the lint passes by mistake).
for layout in \
  "${CHUNKS_DIR}/app/(dashboard)" \
  "${CHUNKS_DIR}/app/popout"; do
  if ! grep -lr 'WasmProvider' "${layout}" >/dev/null 2>&1; then
    echo "FAIL: ${layout} chunks lost WasmProvider reference"
    exit 1
  fi
done

echo "PASS: dashboard / popout layouts retain WasmProvider"
