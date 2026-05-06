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

# `STABLE_RUNNER_VERSION` is what `agentsmesh-runner --version` prints
# at runtime. Identical to `STABLE_IMAGE_VERSION` today (both derived
# from $IMAGE_VERSION) but kept under a distinct key so semantic intent
# is clear: image-tag stamping is for OCI registries; binary stamping
# is for `main.version` injected via `cross_binary(x_defs)`. The two
# could diverge later (e.g. SemVer for the binary, sha-prefix for the
# image) without churning consumers on either side.
build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "STABLE_IMAGE_VERSION ${version}"
echo "STABLE_IMAGE_MINOR ${minor}"
echo "STABLE_RUNNER_VERSION ${version}"
echo "STABLE_BUILD_TIME ${build_time}"
