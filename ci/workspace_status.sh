#!/usr/bin/env bash
# Bazel workspace status script.
#
# Emits "STABLE_*" keys consumed by //build_defs/docker:go_oci_image.bzl's
# expand_template → oci_push tag stamping. Invoked via:
#
#   bazel build --stamp --workspace_status_command=ci/workspace_status.sh ...
#   bazel run   --stamp --workspace_status_command=ci/workspace_status.sh ...
#
# A line `STABLE_KEY value` in stdout becomes a build-time substitution
# pattern `{{STABLE_KEY}}`.
#
# Env inputs (set by CI, optional locally):
#   IMAGE_VERSION  — primary tag (sha-abc, 1.2.3). Falls back to the
#                    short git SHA prefixed with `dev-`.
#   IMAGE_MINOR    — secondary tag (1.2 for semver), defaults to
#                    IMAGE_VERSION (dedup is fine; oci_push tags twice).

set -euo pipefail

version="${IMAGE_VERSION:-}"
if [ -z "${version}" ]; then
  sha="$(git rev-parse --short HEAD 2>/dev/null || echo local)"
  version="dev-${sha}"
fi

minor="${IMAGE_MINOR:-${version}}"

echo "STABLE_IMAGE_VERSION ${version}"
echo "STABLE_IMAGE_MINOR ${minor}"
