#!/usr/bin/env bash
# Identifier contract lint — enforces that raw external strings never flow
# directly into UNIQUE identifier columns. Each rule below targets a known
# anti-pattern from the kudin.private regression. See CLAUDE.md "Identifier
# 字段契约" and backend/pkg/slugkit/doc.go for rationale.
#
# Hits return exit 1; CI gates merges on green.
#
# Modes:
#   default              full repo scan (CI)
#   IDENT_LINT_FAST=1    scan only files changed vs origin/main HEAD..HEAD
#                        (local iteration; falls back to full scan if no diff)
set -euo pipefail

ROOT="${BUILD_WORKSPACE_DIRECTORY:-$(git rev-parse --show-toplevel)}"
cd "$ROOT"

fail=0

# Build the file scope. In fast mode, intersect rule globs with changed
# files; otherwise rules operate on their natural globs.
FAST_MODE=0
CHANGED_FILES=""
if [[ "${IDENT_LINT_FAST:-0}" == "1" ]]; then
  base_ref="${IDENT_LINT_BASE:-origin/main}"
  if git rev-parse --verify "$base_ref" >/dev/null 2>&1; then
    # `origin/main` (no `...`) includes both committed and uncommitted
    # diff vs the base — what local devs actually want before pushing.
    CHANGED_FILES=$(git diff --name-only --diff-filter=ACMR "$base_ref" 2>/dev/null || true)
    if [[ -n "$CHANGED_FILES" ]]; then
      FAST_MODE=1
    fi
  fi
fi

# match_changed filters a newline-separated file list down to files that
# match any of the provided patterns. Returns 0 if any survived.
filter_changed() {
  local pattern="$1"
  if [[ "$FAST_MODE" == "0" ]]; then
    echo "__ALL__"
    return 0
  fi
  echo "$CHANGED_FILES" | grep -E "$pattern" || true
}

violation() {
  local rule="$1"; shift
  local hits="$1"; shift
  if [[ -n "$hits" ]]; then
    echo "❌ identifier-lint: $rule"
    echo "$hits" | sed 's/^/    /'
    fail=1
  fi
}

# scan grep-searches backend/**/*.go for $1. Additional args are passed as
# --exclude=GLOB. In fast mode, the search is narrowed to files in the diff
# matching backend/*.go (excluding _test.go).
scan() {
  local pattern="$1"; shift
  local files
  if [[ "$FAST_MODE" == "1" ]]; then
    files=$(echo "$CHANGED_FILES" | grep -E '^backend/.*\.go$' | grep -vE '_test\.go$' || true)
    if [[ -z "$files" ]]; then
      return 0
    fi
    local exclude_pattern=""
    for glob in "$@"; do
      glob_re="${glob//\*/[^/]*}"
      if [[ -n "$exclude_pattern" ]]; then
        exclude_pattern="$exclude_pattern|"
      fi
      exclude_pattern="$exclude_pattern$glob_re"
    done
    if [[ -n "$exclude_pattern" ]]; then
      files=$(echo "$files" | grep -vE "(^|/)($exclude_pattern)$" || true)
    fi
    if [[ -z "$files" ]]; then
      return 0
    fi
    # shellcheck disable=SC2086
    echo "$files" | xargs grep -nE "$pattern" 2>/dev/null || true
    return 0
  fi
  local cmd=(grep -RnE "$pattern" backend --include='*.go' --exclude='*_test.go')
  if [[ $# -gt 0 ]]; then
    for glob in "$@"; do
      cmd+=(--exclude="$glob")
    done
  fi
  "${cmd[@]}" || true
}

# Rule 1: never derive username via Split(email,"@")[0]; always go through
# userService.EnsureUniqueUsername. Phase 1 removed all known sites.
hits=$(scan 'strings\.Split\([^,]+@[^,]+,\s*"@"\)')
violation "split email-local-part outside helper" "$hits"

# Rule 2: never assign u.Username = <expr> outside username_registry.go and
# the OAuth/SSO funnel — those are the ONLY sanctioned write paths.
# OAuth/SSO parsers (saml_helpers, oidc, oauth_*) raw-passthrough the value;
# user_oauth.go then funnels it through EnsureUniqueUsername.
hits=$(scan '\b[Uu]sername\s*=\s*' \
  'username_registry.go' 'user_oauth.go' 'mock_*.go' 'service_setup_test.go' \
  'saml_helpers.go' 'oidc.go' 'oauth_*.go' 'oauth_providers.go')
hits=$(echo "$hits" | grep -E '\.\s*Username\s*=\s*[^=]' || true)
violation "raw assignment to .Username outside username_registry" "$hits"

# Rule 3: client-side code (Go callers + TS) must not build personal
# workspace slugs by concatenating "-workspace". Server-side
# orgService.CreatePersonal is the only sanctioned generator.
hits=$(scan 'fmt\.Sprintf\([^)]*-workspace' 'service_personal.go')
violation "client-side personal workspace slug concat" "$hits"

# scan_ts grep-searches clients/{web,web-admin,desktop}/src for $1.
# In fast mode, intersects with the diff.
scan_ts() {
  local pattern="$1"
  if [[ "$FAST_MODE" == "1" ]]; then
    local files
    files=$(echo "$CHANGED_FILES" | grep -E '^clients/(web|web-admin|desktop)/src/.*\.(ts|tsx)$' | grep -vE '\.test\.(ts|tsx)$' || true)
    if [[ -z "$files" ]]; then
      return 0
    fi
    # shellcheck disable=SC2086
    echo "$files" | xargs grep -nE "$pattern" 2>/dev/null || true
    return 0
  fi
  grep -RnE "$pattern" \
    clients/web/src clients/web-admin/src clients/desktop/src \
    --include='*.ts' --include='*.tsx' \
    --exclude='*.test.ts' --exclude='*.test.tsx' || true
}

# Rule 4: TypeScript front-end must not build slugs from username either.
ts_hits=$(scan_ts '\$\{[^}]*username[^}]*\}-workspace|user\.username\s*\+\s*['\''"]-workspace')
violation "client-side personal workspace slug concat" "$ts_hits"

# Rule 5: New UNIQUE VARCHAR columns in migrations (>=000135, post-契约
# baseline) must ship with a slug CHECK constraint, OR be a known non-
# identifier column (email, hash, token, etc). Catches the next
# "channels.name was UNIQUE but no format CHECK" anti-pattern. Older
# migrations are grandfathered — applying CHECK retroactively is a
# separate backfill effort tracked elsewhere.
if [[ "$FAST_MODE" == "1" ]]; then
  new_migrations=$(echo "$CHANGED_FILES" | grep -E '^backend/migrations/.*\.up\.sql$' | awk -F/ '{n=$NF; gsub(/_.*/, "", n); if (n+0 >= 135) print $0}' || true)
else
  new_migrations=$(find backend/migrations -name '*.up.sql' -type f 2>/dev/null | awk -F/ '{n=$NF; gsub(/_.*/, "", n); if (n+0 >= 135) print $0}')
fi
for f in $new_migrations; do
  if ! grep -qE 'VARCHAR\([0-9]+\)[^,]*UNIQUE|UNIQUE.*VARCHAR' "$f" 2>/dev/null; then
    continue
  fi
  if grep -qiE 'UNIQUE.*(email|hash|token|url|key_hash|order_no|invoice_no|license_key|external_id)' "$f"; then
    continue
  fi
  if grep -qE 'CHECK[^)]*[a-z0-9]\]?\+.*\(-' "$f" 2>/dev/null || \
     grep -qE 'slug.*format|_username_format|_slug_format' "$f" 2>/dev/null; then
    continue
  fi
  echo "❌ identifier-lint: $f adds UNIQUE column without slug format CHECK"
  fail=1
done

# Rule 6: Frontend route / mention text must not interpolate user.username
# raw. usernames are now slug-compliant (Layer 1-3 enforced), but a wrong
# template like '/u/${user.username}/profile' would still wire identifier
# into UI text — display name (user.name) should be used instead.
ts_route_hits=$(scan_ts 'href=\{?\s*[`"]\s*/[^"`]*\$\{[^}]*\.username\}|push\(\s*[`"]\s*/[^"`]*\$\{[^}]*\.username\}')
violation "user.username used in route path (use org.slug or user.id)" "$ts_route_hits"

if [[ $fail -eq 1 ]]; then
  echo
  echo "See CLAUDE.md '## Identifier 字段契约' for the sanctioned write paths."
  exit 1
fi
mode_tag="full"
[[ "$FAST_MODE" == "1" ]] && mode_tag="fast (diff vs ${IDENT_LINT_BASE:-origin/main})"
echo "✅ identifier-lint: all rules pass [$mode_tag]"
