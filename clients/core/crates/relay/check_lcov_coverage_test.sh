#!/usr/bin/env bash

set -euo pipefail

checker="${TEST_SRCDIR}/${TEST_WORKSPACE}/clients/core/crates/relay/check_lcov_coverage"
tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/agentsmesh-relay-coverage-test.XXXXXX")
trap 'rm -rf "$tmpdir"' EXIT

write_report() {
  local output=$1
  shift
  {
    echo 'TN:'
    printf '%s\n' "$@"
  } >"$output"
}

write_complete_report() {
  local output=$1
  local omitted=${2:-}
  local workspace="${TEST_SRCDIR}/${TEST_WORKSPACE}"
  local source relative base
  {
    echo 'TN:'
    while IFS= read -r source; do
      relative=${source#"$workspace/"}
      base=${relative##*/}
      if [[ "$relative" == */integration_tests/* \
        || "$base" == "integration_tests.rs" \
        || "$base" == *_tests.rs \
        || "$relative" == "$omitted" ]]; then
        continue
      fi
      printf 'SF:%s\nDA:1,1\nend_of_record\n' "$relative"
    done < <(find -L "$workspace/clients/core/crates/relay/src" -type f -name '*.rs' -print | sort)
  } >"$output"
}

expect_exit() {
  local expected=$1
  local stderr_pattern=$2
  shift 2
  local stderr="$tmpdir/stderr"
  local actual=0
  "$@" >/dev/null 2>"$stderr" || actual=$?
  if [[ "$actual" -ne "$expected" ]]; then
    echo "expected exit $expected, got $actual from: $*" >&2
    cat "$stderr" >&2
    exit 1
  fi
  if ! grep -Fq "$stderr_pattern" "$stderr"; then
    echo "missing expected diagnostic '$stderr_pattern' from: $*" >&2
    cat "$stderr" >&2
    exit 1
  fi
}

write_complete_report "$tmpdir/pass.info"
"$checker" "$tmpdir/pass.info" 95 95 >/dev/null

# All per-file floors pass, but the aggregate misses a deliberately stricter
# threshold. This proves aggregate enforcement independently of the file gate.
write_complete_report "$tmpdir/aggregate-below.info"
{
  echo 'SF:clients/core/crates/relay/src/types.rs'
  for line in {2..10}; do
    printf 'DA:%d,1\n' "$line"
  done
  echo 'DA:11,0'
  echo 'end_of_record'
} >>"$tmpdir/aggregate-below.info"
expect_exit 1 \
  'Relay aggregate coverage below threshold:' \
  "$checker" "$tmpdir/aggregate-below.info" 100 95

write_complete_report "$tmpdir/missing-source.info" 'clients/core/crates/relay/src/types.rs'
expect_exit 2 \
  'LCOV is missing relay production source: clients/core/crates/relay/src/types.rs' \
  "$checker" "$tmpdir/missing-source.info" 95 95

# The aggregate remains above 95%, but one non-tiny production file is below
# the 95% floor. This is the regression the per-file ratchet must catch.
write_complete_report "$tmpdir/per-file-below.info"
{
  echo 'SF:clients/core/crates/relay/src/types.rs'
  for line in {2..101}; do
    if (( line <= 89 )); then
      printf 'DA:%d,1\n' "$line"
    else
      printf 'DA:%d,0\n' "$line"
    fi
  done
  echo 'end_of_record'
} >>"$tmpdir/per-file-below.info"
# Add enough covered lines in another production file to keep the aggregate
# above 95%; only the per-file floor should reject this report.
{
  echo 'SF:clients/core/crates/relay/src/connection.rs'
  for line in {2..300}; do
    printf 'DA:%d,1\n' "$line"
  done
  echo 'end_of_record'
} >>"$tmpdir/per-file-below.info"
expect_exit 1 \
  'Relay file coverage below threshold: clients/core/crates/relay/src/types.rs is 88.12%' \
  "$checker" "$tmpdir/per-file-below.info" 95 95

# Tiny production files cannot hide one uncovered line behind percentage
# rounding: at ten executable lines or fewer, the policy requires 100%.
write_complete_report "$tmpdir/tiny-file-below.info"
write_report "$tmpdir/tiny-override.info" \
  'SF:clients/core/crates/relay/src/error.rs' \
  'DA:2,1' \
  'DA:3,1' \
  'DA:4,1' \
  'DA:5,1' \
  'DA:6,1' \
  'DA:7,1' \
  'DA:8,1' \
  'DA:9,0' \
  'end_of_record'
cat "$tmpdir/tiny-override.info" >>"$tmpdir/tiny-file-below.info"
expect_exit 1 \
  'required 100.00% (tiny-file)' \
  "$checker" "$tmpdir/tiny-file-below.info" 95 95

write_complete_report "$tmpdir/no-executable-lines.info" 'clients/core/crates/relay/src/types.rs'
write_report "$tmpdir/no-executable-lines-override.info" \
  'SF:clients/core/crates/relay/src/types.rs' \
  'end_of_record'
cat "$tmpdir/no-executable-lines-override.info" >>"$tmpdir/no-executable-lines.info"
"$checker" "$tmpdir/no-executable-lines.info" 95 95 >"$tmpdir/no-executable-lines.stdout"
grep -Fq \
  'clients/core/crates/relay/src/types.rs (no executable LCOV lines)' \
  "$tmpdir/no-executable-lines.stdout"

write_complete_report "$tmpdir/tests-only.info"
write_report "$tmpdir/tests-only-override.info" \
  'SF:clients/core/crates/relay/src/dispatch_tests.rs' \
  'DA:15,1' \
  'end_of_record'
cat "$tmpdir/tests-only-override.info" >>"$tmpdir/tests-only.info"
"$checker" "$tmpdir/tests-only.info" 95 95 >/dev/null

write_complete_report "$tmpdir/unknown.info"
write_report "$tmpdir/unknown-override.info" \
  'SF:clients/core/crates/relay/src/not_a_source.rs' \
  'DA:1,1' \
  'end_of_record'
cat "$tmpdir/unknown-override.info" >>"$tmpdir/unknown.info"
expect_exit 2 \
  'LCOV references an unknown relay source: clients/core/crates/relay/src/not_a_source.rs' \
  "$checker" "$tmpdir/unknown.info" 95 95

expect_exit 2 \
  'file coverage threshold must be a number between 0 and 100: invalid' \
  "$checker" "$tmpdir/pass.info" 95 invalid
