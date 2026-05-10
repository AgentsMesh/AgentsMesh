#!/usr/bin/env bash
# Image-size regression gate. Fails the test if the input (an OCI tarball
# OR a staged server-directory tree) exceeds the byte limit passed as
# the second argument. Used by `bazel test //clients/web:image_size_gate`
# (and web-admin) to keep image bloat from creeping back in after the
# 2026-05-09 disk-full incident.
#
# Accepts either a regular file (uses byte size) or a directory (uses
# `du -sb`). The OCI tarball (oci_load implicit output) requires a heavy
# `bazel run :image_tarball` to materialise; in CI we point at the
# server-directory tree (`:next_image_server_dir`) instead — it tracks
# the app payload one-to-one and adds a deterministic ~170 MB distroless
# base in production.
set -euo pipefail

usage() { echo "usage: $0 <path> <max-bytes>"; exit 2; }
[[ $# -eq 2 ]] || usage

path=$1
limit=$2

[[ -e $path ]] || { echo "input not found: $path"; exit 2; }

if [[ -d $path || -L $path ]]; then
  # Bazel's runfiles tree exposes directory artifacts as symlinks. `du -L`
  # follows the entry symlink (and any symlink encountered while walking)
  # AND dedupes by inode — important because npm packages chain back into
  # `.pnpm` via symlinks that would otherwise be double-counted by `find +
  # wc -c`. `-k` keeps the unit portable across BSD/GNU; we multiply to
  # bytes for the comparison.
  actual=$(du -sLk "$path" | awk '{print $1*1024}')
else
  actual=$(wc -c <"$path")
fi

human() {
  awk -v b="$1" 'BEGIN{ split("B KB MB GB",u); s=1; while(b>=1024&&s<4){b/=1024;s++} printf "%.1f%s",b,u[s] }'
}

if (( actual > limit )); then
  echo "FAIL: $path is $(human "$actual") ($actual bytes), exceeds $(human "$limit") ($limit bytes)" >&2
  exit 1
fi
echo "OK: $path is $(human "$actual") (limit $(human "$limit"))"
