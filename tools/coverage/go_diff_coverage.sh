#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  go_diff_coverage.sh --lcov FILE --base COMMIT --include PREFIX [OPTIONS]

Checks line coverage for executable Go lines added or modified between the
merge-base of COMMIT and HEAD. Every changed production file must independently
meet the threshold, so a well-covered large file cannot mask an uncovered
small lifecycle owner. PREFIX is a repository-relative file or directory path.
Production *.go files are included; *_test.go files and deleted files are
excluded.

Options:
  --threshold PERCENT       Required changed-line coverage (default: 95).
  --exact-directory         Include only files directly inside PREFIX.
  --exclude-prefix PREFIX   Exclude a file or subtree; may be repeated.
EOF
}

die() {
  echo "go-diff-coverage: error: $*" >&2
  exit 2
}

lcov_file=""
base_ref=""
include_prefix=""
threshold="95"
exact_directory=0
exclude_prefixes=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --lcov)
      [[ $# -ge 2 ]] || die "--lcov requires a value"
      lcov_file="$2"
      shift 2
      ;;
    --base)
      [[ $# -ge 2 ]] || die "--base requires a value"
      base_ref="$2"
      shift 2
      ;;
    --include)
      [[ $# -ge 2 ]] || die "--include requires a value"
      include_prefix="$2"
      shift 2
      ;;
    --threshold)
      [[ $# -ge 2 ]] || die "--threshold requires a value"
      threshold="$2"
      shift 2
      ;;
    --exact-directory)
      exact_directory=1
      shift
      ;;
    --exclude-prefix)
      [[ $# -ge 2 ]] || die "--exclude-prefix requires a value"
      exclude_prefixes[${#exclude_prefixes[@]}]="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

[[ -n "$lcov_file" ]] || die "--lcov is required"
[[ -n "$base_ref" ]] || die "--base is required"
[[ -n "$include_prefix" ]] || die "--include is required"

if ! awk -v value="$threshold" 'BEGIN {
  exit !(value ~ /^[0-9]+([.][0-9]+)?$/ && value + 0 >= 0 && value + 0 <= 100)
}'; then
  die "--threshold must be a number between 0 and 100: $threshold"
fi

invocation_dir="$PWD"
repo_root_candidate="${BUILD_WORKSPACE_DIRECTORY:-}"
if [[ -z "$repo_root_candidate" ]]; then
  repo_root_candidate=$(git rev-parse --show-toplevel 2>/dev/null) || die "not inside a Git repository"
fi
repo_root_logical=$(cd "$repo_root_candidate" && pwd -L)
repo_root=$(cd "$repo_root_candidate" && pwd -P)

case "$lcov_file" in
  /*) ;;
  *) lcov_file="$invocation_dir/$lcov_file" ;;
esac
[[ -r "$lcov_file" ]] || die "LCOV report is not readable: $lcov_file"

normalize_scope_prefix() {
  local option="$1"
  local value="$2"
  value="${value#./}"
  while [[ "$value" == */ ]]; do
    value="${value%/}"
  done
  [[ -n "$value" ]] || value="."
  [[ "$value" != /* ]] || die "$option must be repository-relative"
  if [[ "$value" != "." ]]; then
    case "/$value/" in
      */../*|*/./*) die "$option must not contain '.' or '..' path components" ;;
    esac
  fi
  case "$value" in
    *$'\n'*|*$'\t'*) die "$option contains an unsupported newline or tab" ;;
  esac
  normalized_prefix="$value"
}

normalized_prefix=""
normalize_scope_prefix "--include" "$include_prefix"
include_prefix="$normalized_prefix"
for ((index = 0; index < ${#exclude_prefixes[@]}; index++)); do
  normalize_scope_prefix "--exclude-prefix" "${exclude_prefixes[$index]}"
  exclude_prefixes[index]="$normalized_prefix"
done

git -C "$repo_root" rev-parse --verify "${base_ref}^{commit}" >/dev/null 2>&1 ||
  die "base commit does not exist: $base_ref"
merge_base=$(git -C "$repo_root" merge-base "$base_ref" HEAD 2>/dev/null) ||
  die "base commit has no merge-base with HEAD: $base_ref"

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/go-diff-coverage.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT

status_file="$tmp_dir/name-status"
changed_records="$tmp_dir/changed-records"
changed_files="$tmp_dir/changed-files"
changed_lines="$tmp_dir/changed-lines"
lcov_sources="$tmp_dir/lcov-sources"
lcov_da="$tmp_dir/lcov-da"
missing_sources="$tmp_dir/missing-sources"
uncovered_lines="$tmp_dir/uncovered-lines"
summary_file="$tmp_dir/summary"
: >"$changed_records"
: >"$changed_lines"
: >"$lcov_sources"
: >"$lcov_da"
: >"$uncovered_lines"
: >"$summary_file"

git -C "$repo_root" diff --name-status -z --find-renames "$merge_base" HEAD >"$status_file"

in_scope() {
  local path="$1"
  local remainder
  local excluded

  if [[ "$exact_directory" -eq 1 ]]; then
    if [[ "$include_prefix" == "." ]]; then
      [[ "$path" != */* ]] || return 1
    else
      case "$path" in
        "$include_prefix") ;;
        "$include_prefix"/*)
          remainder="${path#"$include_prefix"/}"
          [[ "$remainder" != */* ]] || return 1
          ;;
        *) return 1 ;;
      esac
    fi
  elif [[ "$include_prefix" != "." ]]; then
    case "$path" in
      "$include_prefix"|"$include_prefix"/*) ;;
      *) return 1 ;;
    esac
  fi

  if [[ ${#exclude_prefixes[@]} -gt 0 ]]; then
    for excluded in "${exclude_prefixes[@]}"; do
      if [[ "$excluded" == "." ]]; then
        return 1
      fi
      case "$path" in
        "$excluded"|"$excluded"/*) return 1 ;;
      esac
    done
  fi
  return 0
}

record_source() {
  local old_path="$1"
  local new_path="$2"
  case "$new_path" in
    *$'\n'*|*$'\t'*) die "changed path contains an unsupported newline or tab: $new_path" ;;
  esac
  if in_scope "$new_path" && [[ "$new_path" == *.go && "$new_path" != *_test.go ]]; then
    printf '%s\t%s\n' "$new_path" "$old_path" >>"$changed_records"
  fi
}

while IFS= read -r -d '' status; do
  case "$status" in
    R*|C*)
      IFS= read -r -d '' old_path || die "malformed Git rename/copy record"
      IFS= read -r -d '' new_path || die "malformed Git rename/copy record"
      record_source "$old_path" "$new_path"
      ;;
    D*)
      IFS= read -r -d '' || die "malformed Git delete record"
      ;;
    *)
      IFS= read -r -d '' path || die "malformed Git change record"
      record_source "$path" "$path"
      ;;
  esac
done <"$status_file"

LC_ALL=C sort -u "$changed_records" -o "$changed_records"
cut -f1 "$changed_records" >"$changed_files"
changed_file_count=$(wc -l <"$changed_files" | tr -d '[:space:]')

if [[ "$changed_file_count" -eq 0 ]]; then
  echo "go-diff-coverage: SKIP [$include_prefix]: no changed production Go sources"
  exit 0
fi

while IFS=$'\t' read -r new_path old_path; do
  per_file_diff="$tmp_dir/file.diff"
  git -C "$repo_root" diff \
    --unified=0 \
    --no-color \
    --no-ext-diff \
    --no-textconv \
    --find-renames \
    "$merge_base" HEAD -- "$old_path" "$new_path" >"$per_file_diff"

  awk -v path="$new_path" '
    /^@@ / {
      range = $0
      sub(/^@@ -[^ ]+ \+/, "", range)
      sub(/ .*/, "", range)
      count = 1
      if (index(range, ",") != 0) {
        split(range, parts, ",")
        start = parts[1] + 0
        count = parts[2] + 0
      } else {
        start = range + 0
      }
      for (offset = 0; offset < count; offset++) {
        print path "\t" start + offset
      }
    }
  ' "$per_file_diff" >>"$changed_lines"
done <"$changed_records"

LC_ALL=C sort -u "$changed_lines" -o "$changed_lines"

awk -v root="$repo_root" -v logical_root="$repo_root_logical" \
  -v source_out="$lcov_sources" -v da_out="$lcov_da" '
  function normalize(path) {
    sub(/\r$/, "", path)
    gsub(/\/+/, "/", path)
    while (substr(path, 1, 2) == "./") {
      path = substr(path, 3)
    }
    if (index(path, root "/") == 1) {
      path = substr(path, length(root) + 2)
    } else if (index(path, logical_root "/") == 1) {
      path = substr(path, length(logical_root) + 2)
    }
    return path
  }
  /^SF:/ {
    source = normalize(substr($0, 4))
    print source > source_out
    next
  }
  /^DA:/ && source != "" {
    data = substr($0, 4)
    sub(/\r$/, "", data)
    split(data, fields, ",")
    print source "\t" fields[1] "\t" fields[2] > da_out
  }
' "$lcov_file"

LC_ALL=C sort -u "$lcov_sources" -o "$lcov_sources"
awk -F '\t' '
  FILENAME == ARGV[1] { present[$0] = 1; next }
  !($1 in present) { print $1 }
' "$lcov_sources" "$changed_records" | LC_ALL=C sort -u >"$missing_sources"

if [[ -s "$missing_sources" ]]; then
  echo "go-diff-coverage: FAIL [$include_prefix]: changed production Go source is absent from LCOV" >&2
  sed 's/^/  missing SF: /' "$missing_sources" >&2
  exit 1
fi

awk -F '\t' -v uncovered_out="$uncovered_lines" -v summary_out="$summary_file" '
  BEGIN { OFS = FS }
  FILENAME == ARGV[1] {
    key = $1 FS $2
    executable[key] = 1
    if ($3 + 0 > hits[key]) {
      hits[key] = $3 + 0
    }
    next
  }
  {
    changed_file[$1] = 1
    key = $1 FS $2
    if (!(key in executable)) {
      next
    }
    eligible[$1]++
    if (hits[key] > 0) {
      covered[$1]++
    } else {
      print $1, $2 > uncovered_out
    }
  }
  END {
    for (file in changed_file) {
      print file, covered[file] + 0, eligible[file] + 0 > summary_out
    }
  }
' "$lcov_da" "$changed_lines"

LC_ALL=C sort -t $'\t' -k1,1 "$summary_file" -o "$summary_file"

overall_covered=0
overall_eligible=0
failed=0
while IFS=$'\t' read -r source covered eligible; do
  [[ -n "$source" ]] || continue
  if [[ "$eligible" -eq 0 ]]; then
    echo "go-diff-coverage: SKIP file [$source]: no changed executable LCOV DA lines"
    continue
  fi

  overall_covered=$((overall_covered + covered))
  overall_eligible=$((overall_eligible + eligible))
  percentage=$(awk -v covered="$covered" -v eligible="$eligible" 'BEGIN {
    printf "%.2f", covered * 100 / eligible
  }')
  threshold_display=$(awk -v threshold="$threshold" 'BEGIN { printf "%.2f", threshold + 0 }')

  if awk -v covered="$covered" -v eligible="$eligible" -v threshold="$threshold" 'BEGIN {
    exit !(covered * 100 >= eligible * threshold)
  }'; then
    echo "go-diff-coverage: PASS file [$source]: ${percentage}% (${covered}/${eligible}), threshold ${threshold_display}%"
    continue
  fi

  failed=1
  echo "go-diff-coverage: FAIL file [$source]: ${percentage}% (${covered}/${eligible}), below ${threshold_display}%" >&2
  echo "  uncovered changed executable lines:" >&2
  awk -F '\t' -v source="$source" '$1 == source { print "    " $1 ":" $2 }' \
    "$uncovered_lines" | sed -n '1,50p' >&2
  file_uncovered_count=$(awk -F '\t' -v source="$source" '$1 == source { count++ } END { print count + 0 }' \
    "$uncovered_lines")
  if [[ "$file_uncovered_count" -gt 50 ]]; then
    echo "    ... and $((file_uncovered_count - 50)) more" >&2
  fi
done <"$summary_file"

if [[ "$overall_eligible" -eq 0 ]]; then
  echo "go-diff-coverage: SKIP [$include_prefix]: $changed_file_count changed production Go source(s), but no changed executable LCOV DA lines"
  exit 0
fi

overall_percentage=$(awk -v covered="$overall_covered" -v eligible="$overall_eligible" 'BEGIN {
  printf "%.2f", covered * 100 / eligible
}')
threshold_display=$(awk -v threshold="$threshold" 'BEGIN { printf "%.2f", threshold + 0 }')

if [[ "$failed" -eq 0 ]]; then
  echo "go-diff-coverage: PASS [$include_prefix]: every executable changed file is >= ${threshold_display}% (aggregate ${overall_percentage}%, ${overall_covered}/${overall_eligible})"
  exit 0
fi

echo "go-diff-coverage: FAIL [$include_prefix]: at least one changed production file is below ${threshold_display}% (aggregate ${overall_percentage}%, ${overall_covered}/${overall_eligible})" >&2
exit 1
