#!/usr/bin/env bash
# shellcheck disable=SC1091,SC2034,SC2154

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

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=go_diff_coverage_git.sh
source "$script_dir/go_diff_coverage_git.sh"
# shellcheck source=go_diff_coverage_lcov.sh
source "$script_dir/go_diff_coverage_lcov.sh"

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
    *) die "unknown argument: $1" ;;
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

normalize_scope_configuration
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

collect_changed_go_files
if [[ "$changed_file_count" -eq 0 ]]; then
  echo "go-diff-coverage: SKIP [$include_prefix]: no changed production Go sources"
  exit 0
fi
collect_changed_go_lines
check_changed_go_coverage
