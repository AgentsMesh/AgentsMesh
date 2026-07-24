#!/usr/bin/env bash
# shellcheck disable=SC2034,SC2154

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

normalize_scope_configuration() {
  normalized_prefix=""
  normalize_scope_prefix "--include" "$include_prefix"
  include_prefix="$normalized_prefix"
  for ((index = 0; index < ${#exclude_prefixes[@]}; index++)); do
    normalize_scope_prefix "--exclude-prefix" "${exclude_prefixes[$index]}"
    exclude_prefixes[index]="$normalized_prefix"
  done
}

in_scope() {
  local path="$1"
  local remainder excluded

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
      [[ "$excluded" != "." ]] || return 1
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

collect_changed_go_files() {
  git -C "$repo_root" diff --name-status -z --find-renames "$merge_base" HEAD >"$status_file"

  local status old_path new_path path
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
}

collect_changed_go_lines() {
  local new_path old_path per_file_diff
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
}
