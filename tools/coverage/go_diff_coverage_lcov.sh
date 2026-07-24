#!/usr/bin/env bash
# shellcheck disable=SC2034,SC2154

parse_lcov_report() {
  awk -v root="$repo_root" -v logical_root="$repo_root_logical" \
    -v source_out="$lcov_sources" -v da_out="$lcov_da" '
    function normalize(path) {
      sub(/\r$/, "", path)
      gsub(/\/+/, "/", path)
      while (substr(path, 1, 2) == "./") path = substr(path, 3)
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
}

require_changed_sources_in_lcov() {
  awk -F '\t' '
    FILENAME == ARGV[1] { present[$0] = 1; next }
    !($1 in present) { print $1 }
  ' "$lcov_sources" "$changed_records" | LC_ALL=C sort -u >"$missing_sources"

  if [[ -s "$missing_sources" ]]; then
    echo "go-diff-coverage: FAIL [$include_prefix]: changed production Go source is absent from LCOV" >&2
    sed 's/^/  missing SF: /' "$missing_sources" >&2
    return 1
  fi
}

summarize_changed_line_coverage() {
  awk -F '\t' -v uncovered_out="$uncovered_lines" -v summary_out="$summary_file" '
    BEGIN { OFS = FS }
    FILENAME == ARGV[1] {
      key = $1 FS $2
      executable[key] = 1
      if ($3 + 0 > hits[key]) hits[key] = $3 + 0
      next
    }
    {
      changed_file[$1] = 1
      key = $1 FS $2
      if (!(key in executable)) next
      eligible[$1]++
      if (hits[key] > 0) covered[$1]++
      else print $1, $2 > uncovered_out
    }
    END {
      for (file in changed_file) {
        print file, covered[file] + 0, eligible[file] + 0 > summary_out
      }
    }
  ' "$lcov_da" "$changed_lines"

  LC_ALL=C sort -t $'\t' -k1,1 "$summary_file" -o "$summary_file"
}

report_changed_line_coverage() {
  local overall_covered=0 overall_eligible=0 failed=0
  local source covered eligible percentage threshold_display file_uncovered_count
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
    return 0
  fi

  local overall_percentage
  overall_percentage=$(awk -v covered="$overall_covered" -v eligible="$overall_eligible" 'BEGIN {
    printf "%.2f", covered * 100 / eligible
  }')
  threshold_display=$(awk -v threshold="$threshold" 'BEGIN { printf "%.2f", threshold + 0 }')

  if [[ "$failed" -eq 0 ]]; then
    echo "go-diff-coverage: PASS [$include_prefix]: every executable changed file is >= ${threshold_display}% (aggregate ${overall_percentage}%, ${overall_covered}/${overall_eligible})"
    return 0
  fi

  echo "go-diff-coverage: FAIL [$include_prefix]: at least one changed production file is below ${threshold_display}% (aggregate ${overall_percentage}%, ${overall_covered}/${overall_eligible})" >&2
  return 1
}

check_changed_go_coverage() {
  parse_lcov_report
  require_changed_sources_in_lcov
  summarize_changed_line_coverage
  report_changed_line_coverage
}
