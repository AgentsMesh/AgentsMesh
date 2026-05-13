#!/usr/bin/env bash
# Verify that marketing-page chunks don't pull in wasm. Run after
# `bazel build //clients/web:next`. Exits non-zero if any leak.
#
# Architectural invariant (Plan I1):
#   Routes outside (dashboard) / (auth) / popout / blocks-embed must not
#   load 21MB of wasm — they're either static / public-API-only.

set -euo pipefail

# `:next` writes to `.next-dev/` (see clients/web/BUILD.bazel and
# next.config.ts for why). Allow override for callers that already
# point NEXT_DIR at a custom location.
NEXT_DIR="${NEXT_DIR:-bazel-bin/clients/web/.next-dev}"
CHUNKS_DIR="${NEXT_DIR}/static/chunks"

if [[ ! -d "${CHUNKS_DIR}" ]]; then
  echo "FAIL: ${CHUNKS_DIR} not found. Run: bazel build //clients/web:next"
  exit 2
fi

# Marketing route chunks: must not contain WasmProvider / initWasmCore /
# any wasm-bindgen class names.
LEAKS=$(
  find "${CHUNKS_DIR}/app" -maxdepth 4 -name "*.js" \
    -not -path "*\(dashboard\)*" \
    -not -path "*\(auth\)*" \
    -not -path "*popout*" \
    -not -path "*blocks-embed*" \
    -print0 \
  | xargs -0 grep -l 'WasmProvider\|initWasmCore\|WasmApiClient\|WasmAuthManager\|wasm_pkg\|agentsmesh-wasm' 2>/dev/null \
  || true
)

if [[ -n "${LEAKS}" ]]; then
  echo "FAIL: marketing chunks contain wasm symbols:"
  echo "${LEAKS}" | sed 's/^/  /'
  exit 1
fi

echo "PASS: no wasm symbols in marketing chunks"

# Confirm wasm-bound layouts DO contain WasmProvider (positive check —
# catches a future regression where someone deletes the (auth)/(dashboard)
# WasmProvider import and the lint passes by mistake).
for layout in \
  "${CHUNKS_DIR}/app/(dashboard)" \
  "${CHUNKS_DIR}/app/(auth)" \
  "${CHUNKS_DIR}/app/popout"; do
  if ! grep -lr 'WasmProvider' "${layout}" >/dev/null 2>&1; then
    echo "FAIL: ${layout} chunks lost WasmProvider reference"
    exit 1
  fi
done

echo "PASS: dashboard / auth / popout layouts retain WasmProvider"
