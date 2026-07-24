#!/usr/bin/env bash

set -euo pipefail

workspace="${TEST_WORKSPACE:-_main}"
gate="${TEST_SRCDIR:-}/$workspace/tools/coverage/go_diff_coverage.sh"
if [[ ! -x "$gate" ]]; then
  gate="$(cd "$(dirname "$0")" && pwd)/go_diff_coverage.sh"
fi
[[ -x "$gate" ]] || {
  echo "go_diff_coverage.sh runfile not found" >&2
  exit 1
}

if [[ -n "${TEST_TMPDIR:-}" ]]; then
  tmp_root="$TEST_TMPDIR"
else
  tmp_root=$(mktemp -d "${TMPDIR:-/tmp}/go-diff-coverage-test.XXXXXX")
  trap 'rm -rf "$tmp_root"' EXIT
fi
repo="$tmp_root/repo"
mkdir -p "$repo"
git -C "$repo" init -q
git -C "$repo" config user.email coverage@example.com
git -C "$repo" config user.name "Coverage Test"

write_sample() {
  local value="$1"
  mkdir -p "$repo/runner/internal/runner"
  {
    printf '%s\n' 'package runner' '' 'func sampleValue() int {'
    printf '\treturn %s\n' "$value"
    printf '%s\n' '}'
  } >"$repo/runner/internal/runner/sample.go"
}

write_lcov() {
  local destination="$1"
  local source="$2"
  local return_hits="$3"
  printf '%s\n' \
    'TN:' \
    "SF:$source" \
    'DA:3,1' \
    "DA:4,$return_hits" \
    'DA:5,1' \
    'end_of_record' >"$destination"
}

run_gate_scope() {
  local report="$1"
  local base="$2"
  local include="$3"
  shift 3
  BUILD_WORKSPACE_DIRECTORY="$repo" "$gate" \
    --lcov "$report" \
    --base "$base" \
    --include "$include" \
    --threshold 95 \
    "$@"
}

run_gate() {
  run_gate_scope "$1" "$2" runner/internal/runner
}

expect_pass() {
  local label="$1"
  local expected="$2"
  shift 2
  local output="$tmp_root/$label.out"
  if ! "$@" >"$output" 2>&1; then
    echo "$label: expected success" >&2
    cat "$output" >&2
    exit 1
  fi
  if ! grep -F "$expected" "$output" >/dev/null; then
    echo "$label: output did not contain: $expected" >&2
    cat "$output" >&2
    exit 1
  fi
}

expect_fail() {
  local label="$1"
  local expected="$2"
  shift 2
  local output="$tmp_root/$label.out"
  if "$@" >"$output" 2>&1; then
    echo "$label: expected failure" >&2
    cat "$output" >&2
    exit 1
  fi
  if ! grep -F "$expected" "$output" >/dev/null; then
    echo "$label: output did not contain: $expected" >&2
    cat "$output" >&2
    exit 1
  fi
}

write_sample 1
git -C "$repo" add .
git -C "$repo" commit -qm baseline
base=$(git -C "$repo" rev-parse HEAD)

write_sample 2
git -C "$repo" add .
git -C "$repo" commit -qm changed

pass_lcov="$tmp_root/pass.lcov"
write_lcov "$pass_lcov" "$repo/runner/internal/runner/sample.go" 1
expect_pass pass '100.00% (1/1)' run_gate "$pass_lcov" "$base"

below_lcov="$tmp_root/below.lcov"
write_lcov "$below_lcov" runner/internal/runner/sample.go 0
expect_fail below 'below 95.00%' run_gate "$below_lcov" "$base"
expect_fail below-line 'runner/internal/runner/sample.go:4' run_gate "$below_lcov" "$base"

missing_lcov="$tmp_root/missing.lcov"
printf '%s\n' 'TN:' 'end_of_record' >"$missing_lcov"
expect_fail missing 'missing SF: runner/internal/runner/sample.go' run_gate "$missing_lcov" "$base"

zero_da_lcov="$tmp_root/zero-da.lcov"
printf '%s\n' \
  'TN:' \
  'SF:runner/internal/runner/sample.go' \
  'end_of_record' >"$zero_da_lcov"
expect_pass zero-da 'no changed executable LCOV DA lines' run_gate "$zero_da_lcov" "$base"

comment_base=$(git -C "$repo" rev-parse HEAD)
printf '%s\n' \
  'package runner' \
  '' \
  '// sampleValue returns the current fixture value.' \
  'func sampleValue() int {' \
  $'\treturn 2' \
  '}' >"$repo/runner/internal/runner/sample.go"
git -C "$repo" add .
git -C "$repo" commit -qm comment-only
comment_lcov="$tmp_root/comment.lcov"
printf '%s\n' \
  'TN:' \
  'SF:runner/internal/runner/sample.go' \
  'DA:4,1' \
  'DA:5,1' \
  'DA:6,1' \
  'end_of_record' >"$comment_lcov"
expect_pass comment-only 'no changed executable LCOV DA lines' run_gate "$comment_lcov" "$comment_base"
expect_fail comment-missing 'missing SF: runner/internal/runner/sample.go' run_gate "$missing_lcov" "$comment_base"

printf '%s\n' \
  'package runner' \
  '' \
  'func obsoleteValue() int {' \
  $'\treturn 1' \
  '}' >"$repo/runner/internal/runner/deletion_only.go"
git -C "$repo" add .
git -C "$repo" commit -qm deletion-baseline
deletion_base=$(git -C "$repo" rev-parse HEAD)
printf '%s\n' 'package runner' >"$repo/runner/internal/runner/deletion_only.go"
git -C "$repo" add .
git -C "$repo" commit -qm deletion-only
deletion_lcov="$tmp_root/deletion.lcov"
printf '%s\n' \
  'TN:' \
  'SF:runner/internal/runner/deletion_only.go' \
  'end_of_record' >"$deletion_lcov"
expect_pass deletion-only 'no changed executable LCOV DA lines' \
  run_gate "$deletion_lcov" "$deletion_base"

no_change_base=$(git -C "$repo" rev-parse HEAD)
mkdir -p "$repo/docs"
printf '%s\n' 'coverage documentation' >"$repo/docs/coverage.md"
git -C "$repo" add .
git -C "$repo" commit -qm docs
expect_pass no-change 'SKIP [runner/internal/runner]' run_gate "$pass_lcov" "$no_change_base"

test_only_base=$(git -C "$repo" rev-parse HEAD)
printf '%s\n' \
  'package runner' \
  '' \
  'import "testing"' \
  '' \
  'func TestSample(t *testing.T) {}' >"$repo/runner/internal/runner/sample_test.go"
git -C "$repo" add .
git -C "$repo" commit -qm test-only
expect_pass test-only 'no changed production Go sources' run_gate "$pass_lcov" "$test_only_base"

rename_base=$(git -C "$repo" rev-parse HEAD)
git -C "$repo" mv runner/internal/runner/sample.go runner/internal/runner/renamed.go
sed 's/return 2/return 3/' "$repo/runner/internal/runner/renamed.go" >"$tmp_root/renamed.go"
mv "$tmp_root/renamed.go" "$repo/runner/internal/runner/renamed.go"
git -C "$repo" add .
git -C "$repo" commit -qm rename
rename_lcov="$tmp_root/rename.lcov"
write_lcov "$rename_lcov" runner/internal/runner/renamed.go 1
expect_pass rename '100.00% (1/1)' run_gate "$rename_lcov" "$rename_base"

new_file_base=$(git -C "$repo" rev-parse HEAD)
printf '%s\n' \
  'package runner' \
  '' \
  'func newValue() int {' \
  $'\treturn 1' \
  '}' >"$repo/runner/internal/runner/new_file.go"
git -C "$repo" add .
git -C "$repo" commit -qm new-file
new_file_lcov="$tmp_root/new-file.lcov"
write_lcov "$new_file_lcov" runner/internal/runner/new_file.go 1
expect_pass new-file '100.00% (3/3)' run_gate "$new_file_lcov" "$new_file_base"

# A module-level aggregate of 20/21 is above 95%, but the uncovered one-line
# owner must still fail independently. This prevents a large, well-covered
# refactor from masking an untested lifecycle adapter in the same package.
write_mask_large() {
  local value="$1"
  {
    printf '%s\n' 'package runner' '' 'func maskLarge() int {'
    for _ in {1..20}; do
      printf '\t_ = %s\n' "$value"
    done
    printf '%s\n' $'\treturn 1' '}'
  } >"$repo/runner/internal/runner/mask_large.go"
}

write_mask_large 1
printf '%s\n' \
  'package runner' \
  '' \
  'func maskSmall() int {' \
  $'\treturn 1' \
  '}' >"$repo/runner/internal/runner/mask_small.go"
git -C "$repo" add .
git -C "$repo" commit -qm mask-baseline
mask_base=$(git -C "$repo" rev-parse HEAD)

write_mask_large 2
sed 's/return 1/return 2/' "$repo/runner/internal/runner/mask_small.go" >"$tmp_root/mask-small.go"
mv "$tmp_root/mask-small.go" "$repo/runner/internal/runner/mask_small.go"
git -C "$repo" add .
git -C "$repo" commit -qm mask-changes
mask_lcov="$tmp_root/mask.lcov"
{
  printf '%s\n' 'TN:' 'SF:runner/internal/runner/mask_large.go'
  for line in {4..23}; do
    printf 'DA:%d,1\n' "$line"
  done
  printf '%s\n' 'end_of_record' 'SF:runner/internal/runner/mask_small.go' \
    'DA:4,0' 'end_of_record'
} >"$mask_lcov"
expect_fail per-file-mask 'FAIL file [runner/internal/runner/mask_small.go]: 0.00% (0/1)' \
  run_gate "$mask_lcov" "$mask_base"
expect_fail per-file-mask-aggregate 'aggregate 95.24%, 20/21' \
  run_gate "$mask_lcov" "$mask_base"

write_scoped_file() {
  local path="$1"
  local function_name="$2"
  local value="$3"
  mkdir -p "$(dirname "$repo/$path")"
  {
    printf '%s\n' 'package scoped' '' "func $function_name() int {"
    printf '\treturn %s\n' "$value"
    printf '%s\n' '}'
  } >"$repo/$path"
}

write_scoped_file runner/internal/terminal/core.go coreValue 1
write_scoped_file runner/internal/terminal/vt/vt.go vtValue 1
write_scoped_file runner/internal/terminal/aggregator/aggregator.go aggregatorValue 1
git -C "$repo" add .
git -C "$repo" commit -qm scope-baseline
scope_base=$(git -C "$repo" rev-parse HEAD)

write_scoped_file runner/internal/terminal/core.go coreValue 2
write_scoped_file runner/internal/terminal/vt/vt.go vtValue 2
write_scoped_file runner/internal/terminal/aggregator/aggregator.go aggregatorValue 2
git -C "$repo" add .
git -C "$repo" commit -qm scope-changes
scope_lcov="$tmp_root/scope.lcov"
printf '%s\n' \
  'TN:' \
  'SF:runner/internal/terminal/core.go' \
  'DA:3,1' 'DA:4,1' 'DA:5,1' \
  'end_of_record' \
  'SF:runner/internal/terminal/vt/vt.go' \
  'DA:3,1' 'DA:4,0' 'DA:5,1' \
  'end_of_record' \
  'SF:runner/internal/terminal/aggregator/aggregator.go' \
  'DA:3,1' 'DA:4,0' 'DA:5,1' \
  'end_of_record' >"$scope_lcov"

expect_fail recursive-scope 'aggregate 33.33%, 1/3' \
  run_gate_scope "$scope_lcov" "$scope_base" runner/internal/terminal
expect_pass exact-directory '100.00% (1/1)' \
  run_gate_scope "$scope_lcov" "$scope_base" runner/internal/terminal --exact-directory
expect_pass excluded-prefixes '100.00% (1/1)' \
  run_gate_scope "$scope_lcov" "$scope_base" runner/internal/terminal \
  --exclude-prefix runner/internal/terminal/vt \
  --exclude-prefix runner/internal/terminal/aggregator

nested_only_base=$(git -C "$repo" rev-parse HEAD)
write_scoped_file runner/internal/terminal/vt/vt.go vtValue 3
git -C "$repo" add .
git -C "$repo" commit -qm nested-only
expect_pass exact-nested-only 'no changed production Go sources' \
  run_gate_scope "$missing_lcov" "$nested_only_base" runner/internal/terminal --exact-directory
expect_pass excluded-nested-only 'no changed production Go sources' \
  run_gate_scope "$missing_lcov" "$nested_only_base" runner/internal/terminal \
  --exclude-prefix runner/internal/terminal/vt

write_scoped_file runner/internal/agents/mockagent/resize_other.go resizeFallback 1
git -C "$repo" add .
git -C "$repo" commit -qm platform-baseline
platform_base=$(git -C "$repo" rev-parse HEAD)
write_scoped_file runner/internal/agents/mockagent/resize_other.go resizeFallback 2
git -C "$repo" add .
git -C "$repo" commit -qm platform-change
expect_fail platform-source-missing \
  'missing SF: runner/internal/agents/mockagent/resize_other.go' \
  run_gate_scope "$missing_lcov" "$platform_base" runner/internal/agents/mockagent
expect_pass platform-file-excluded 'no changed production Go sources' \
  run_gate_scope "$missing_lcov" "$platform_base" runner/internal/agents/mockagent \
  --exclude-prefix runner/internal/agents/mockagent/resize_other.go

echo "go_diff_coverage_test: all cases passed"
