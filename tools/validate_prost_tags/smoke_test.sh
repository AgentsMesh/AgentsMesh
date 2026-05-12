#!/usr/bin/env bash
# Smoke test for tools/validate_prost_tags. Verifies the validator
# returns 0 for the matched .proto/.rs pair and a non-zero exit + a
# diagnostic line for the drifted pair.
set -euo pipefail

validator="$(dirname "$0")/validate_prost_tags"
testdata="$(dirname "$0")/testdata"

# Bazel runfiles indirection: when sh_test runs us, the validator binary
# and testdata sit under the same runfiles tree. Use realpath to follow
# the symlink Bazel sets up.
good_proto="$testdata/good.proto"
good_rs="$testdata/good.rs"
bad_proto="$testdata/bad.proto"
bad_rs="$testdata/bad.rs"

echo "case 1: matched proto + rust → expect exit 0"
"$validator" "$good_proto" "$good_rs"
echo "  ok"

echo "case 2: drifted rust (tag swap) → expect exit 1 + diagnostic"
if "$validator" "$bad_proto" "$bad_rs" 2>/tmp/validate_prost_tags_stderr; then
    echo "FAIL: validator should have detected the drift" >&2
    exit 1
fi
if ! grep -q 'name:.*tag=2.*tag=3' /tmp/validate_prost_tags_stderr; then
    echo "FAIL: expected diagnostic about field 'name'; got:" >&2
    cat /tmp/validate_prost_tags_stderr >&2
    exit 1
fi
echo "  ok (diagnostic: $(head -1 /tmp/validate_prost_tags_stderr))"
echo "validator smoke test passed."
