#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <combined-lcov-file> [minimum-aggregate-line-percent] [minimum-file-line-percent]" >&2
  exit 2
}

[[ $# -ge 1 && $# -le 3 ]] || usage

report=$1
aggregate_threshold=${2:-95}
file_threshold=${3:-95}
[[ -f "$report" ]] || {
  echo "coverage report does not exist: $report" >&2
  exit 2
}
for threshold_name in aggregate file; do
  if [[ "$threshold_name" == aggregate ]]; then
    threshold=$aggregate_threshold
  else
    threshold=$file_threshold
  fi
  awk -v value="$threshold" 'BEGIN {
    if (value !~ /^[0-9]+([.][0-9]+)?$/ || value < 0 || value > 100) exit 1
  }' || {
    echo "$threshold_name coverage threshold must be a number between 0 and 100: $threshold" >&2
    exit 2
  }
done

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
if [[ -n "${TEST_SRCDIR:-}" && -n "${TEST_WORKSPACE:-}" && -d "${TEST_SRCDIR}/${TEST_WORKSPACE}/clients/core/crates/relay/src" ]]; then
  workspace="${TEST_SRCDIR}/${TEST_WORKSPACE}"
elif [[ -n "${BUILD_WORKSPACE_DIRECTORY:-}" ]]; then
  workspace=$BUILD_WORKSPACE_DIRECTORY
else
  workspace=$(cd "$script_dir/../../../.." && pwd)
fi
source_root="$workspace/clients/core/crates/relay/src"
[[ -d "$source_root" ]] || {
  echo "relay source directory does not exist: $source_root" >&2
  exit 2
}

manifest=$(mktemp "${TMPDIR:-/tmp}/agentsmesh-relay-coverage.XXXXXX")
trap 'rm -f "$manifest"' EXIT

while IFS= read -r source; do
  relative=${source#"$workspace/"}
  printf 'F\t%s\n' "$relative" >>"$manifest"
  awk -v relative="$relative" '
    /LCOV_EXCL_START: test-only code/ {
      if (inside) {
        printf "%s:%d: nested LCOV_EXCL_START\n", relative, NR > "/dev/stderr"
        invalid = 1
      }
      inside = 1
      start = NR
    }
    /^[[:space:]]*#\[cfg\(.*test/ && !inside {
      printf "%s:%d: cfg(test) must be inside explicit LCOV exclusion markers\n", relative, NR > "/dev/stderr"
      invalid = 1
    }
    /LCOV_EXCL_STOP/ {
      if (!inside) {
        printf "%s:%d: LCOV_EXCL_STOP without a start marker\n", relative, NR > "/dev/stderr"
        invalid = 1
      } else {
        printf "X\t%s\t%d\t%d\n", relative, start, NR
        inside = 0
      }
    }
    END {
      if (inside) {
        printf "%s:%d: unterminated LCOV exclusion marker\n", relative, start > "/dev/stderr"
        invalid = 1
      }
      exit invalid
    }
  ' "$source" >>"$manifest"
done < <(find -L "$source_root" -type f -name '*.rs' -print | sort)

# A missed line in a tiny adapter can represent its entire failure path. Such
# files require full coverage instead of being rounded through the 95% floor.
awk -F '\t' \
  -v prefix='clients/core/crates/relay/src/' \
  -v aggregate_threshold="$aggregate_threshold" \
  -v file_threshold="$file_threshold" \
  -v tiny_file_max_lines=10 '
  NR == FNR {
    if ($1 == "F") {
      known[$2] = 1
      if (!is_separate_test_file($2)) {
        production_files[++production_file_count] = $2
      }
    } else if ($1 == "X") {
      for (line = $3; line <= $4; line++) {
        excluded[$2 SUBSEP line] = 1
      }
    }
    next
  }

  function normalized_source(raw, position) {
    position = index(raw, prefix)
    return position ? substr(raw, position) : ""
  }

  function is_separate_test_file(path, base) {
    base = path
    sub(/^.*\//, "", base)
    return path ~ /\/integration_tests\// ||
      base == "integration_tests.rs" ||
      base ~ /_tests\.rs$/
  }

  /^SF:/ {
    source = normalized_source(substr($0, 4))
    whole_file_excluded = source != "" && is_separate_test_file(source)
    if (source != "") {
      seen[source] = 1
      if (!(source in known)) {
        printf "LCOV references an unknown relay source: %s\n", source > "/dev/stderr"
        invalid = 1
      }
    }
    next
  }

  /^DA:/ && source != "" {
    split(substr($0, 4), fields, ",")
    line_number = fields[1] + 0
    hit_count = fields[2] + 0
    if (whole_file_excluded || excluded[source SUBSEP line_number]) {
      next
    }
    key = source SUBSEP line_number
    measured[key] = 1
    source_for[key] = source
    if (hit_count > hits[key]) {
      hits[key] = hit_count
    }
    next
  }

  END {
    for (i = 1; i <= production_file_count; i++) {
      file = production_files[i]
      if (!(file in seen)) {
        printf "LCOV is missing relay production source: %s\n", file > "/dev/stderr"
        invalid = 1
      }
    }
    if (invalid) exit 2

    for (key in measured) {
      total++
      file = source_for[key]
      file_total[file]++
      if (hits[key] > 0) {
        covered++
        file_covered[file]++
      }
    }
    for (i = 1; i <= production_file_count; i++) {
      if (file_total[production_files[i]] > 0) {
        measured_file_count++
      }
    }
    if (total == 0) {
      print "coverage check found zero relay production executable lines" > "/dev/stderr"
      exit 2
    }

    percentage = covered * 100.0 / total
    printf "Relay aggregate production line coverage: %.2f%% (%d/%d across %d executable files); threshold: %.2f%%\n", \
      percentage, covered, total, measured_file_count, aggregate_threshold

    for (i = 1; i <= production_file_count; i++) {
      file = production_files[i]
      if (file_total[file] == 0) {
        printf "     SKIP             %s (no executable LCOV lines)\n", file
        continue
      }
      file_percentage = file_covered[file] * 100.0 / file_total[file]
      required = file_total[file] <= tiny_file_max_lines ? 100 : file_threshold
      policy = file_total[file] <= tiny_file_max_lines ? "tiny-file" : "per-file"
      printf "  %6.2f%%  %4d/%-4d  %s (required: %.2f%%, %s)\n", \
        file_percentage, file_covered[file], file_total[file], file, required, policy
      if (file_percentage + 0.0000001 < required) {
        printf "Relay file coverage below threshold: %s is %.2f%% (%d/%d); required %.2f%% (%s)\n", \
          file, file_percentage, file_covered[file], file_total[file], required, policy > "/dev/stderr"
        file_failed = 1
      }
    }
    if (percentage + 0.0000001 < aggregate_threshold) {
      printf "Relay aggregate coverage below threshold: %.2f%% (%d/%d); required %.2f%%\n", \
        percentage, covered, total, aggregate_threshold > "/dev/stderr"
      aggregate_failed = 1
    }
    if (aggregate_failed || file_failed) exit 1
  }
' "$manifest" "$report"
