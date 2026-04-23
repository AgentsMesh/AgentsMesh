"""OCI image macro for AgentsMesh Go services.

Produces an OCI image (distroless-based), a local `_tarball` for
`docker load`, and one `_push_<key>` target per registry in
`repositories`. Tag stamping plugs into `ci/workspace_status.sh` via
Bazel's `--stamp --workspace_status_command=...` flags so the tag list
comes from the CI environment, not the BUILD file.

Usage:
    load("//build_defs/docker:go_oci_image.bzl", "go_oci_image")

    go_oci_image(
        name = "image",
        binary = ":server",
        exposed_ports = ["8080/tcp"],
        repositories = {
            "dockerhub":  "agentsmesh/backend",
            "agentsmesh": "registry.agentsmesh.ai/agentsmesh/backend",
        },
    )

    # Produces:
    #   :image                   — oci_image
    #   :image_tarball           — oci_load (docker load)
    #   :image_push_dockerhub    — oci_push to Docker Hub
    #   :image_push_agentsmesh   — oci_push to AgentsMesh registry

Replaces: ci/backend.Dockerfile, ci/runner.Dockerfile, ci/relay.Dockerfile.
"""

load("@aspect_bazel_lib//lib:expand_template.bzl", "expand_template")
load("@rules_oci//oci:defs.bzl", "oci_image", "oci_load", "oci_push")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")

def go_oci_image(
        name,
        binary,
        base = "@distroless_static",
        env = {},
        exposed_ports = [],
        labels = {},
        repositories = {},
        repository = None,
        visibility = ["//visibility:public"]):
    """Bundle a Go binary into a distroless OCI image + push targets.

    Args:
        name: Target name.
        binary: Label of the `go_binary` to embed.
        base: OCI base image, defaults to distroless/static.
        env: Environment variables inside the container.
        exposed_ports: Ports to expose on the image.
        labels: OCI labels to add to the image.
        repositories: Dict mapping registry key → full repo URL. Every
            entry becomes a `:name_push_<key>` target. `bazel run` one
            of them with `--stamp --workspace_status_command=...` to
            push with the CI-provided version tag.
        repository: Back-compat single-registry form; equivalent to
            `repositories = {"default": repository}`. Prefer the dict.
        visibility: Standard visibility.
    """
    binary_name = binary.split(":")[-1]

    pkg_tar(
        name = name + "_layer",
        srcs = [binary],
        package_dir = "/app",
    )

    oci_image(
        name = name,
        base = base,
        entrypoint = ["/app/" + binary_name],
        env = env,
        exposed_ports = exposed_ports,
        labels = labels,
        tars = [":" + name + "_layer"],
        user = "nonroot",
        visibility = visibility,
        workdir = "/app",
    )

    oci_load(
        name = name + "_tarball",
        image = ":" + name,
        repo_tags = ["agentsmesh/" + binary_name + ":latest"],
        visibility = visibility,
    )

    # Normalize single-repo back-compat form.
    repos = dict(repositories)
    if repository and not repos:
        repos["default"] = repository

    if repos:
        # Tag list: always `latest`, plus the stamped version + minor.
        # Duplicate values are harmless — oci_push de-dups tags and the
        # registry just re-tags the manifest.
        expand_template(
            name = name + "_tags",
            out = name + "_tags.txt",
            template = [
                "latest",
                "STAMP_VERSION",
                "STAMP_MINOR",
            ],
            stamp_substitutions = {
                "STAMP_VERSION": "{{STABLE_IMAGE_VERSION}}",
                "STAMP_MINOR": "{{STABLE_IMAGE_MINOR}}",
            },
        )

        for reg_key in sorted(repos.keys()):
            oci_push(
                name = "{}_push_{}".format(name, reg_key),
                image = ":" + name,
                remote_tags = ":" + name + "_tags",
                repository = repos[reg_key],
                visibility = visibility,
            )

        # Back-compat alias — first registry's push under the legacy
        # `:name_push` name. Pick `default` if present, else the first
        # alphabetical key.
        back_compat_key = "default" if "default" in repos else sorted(repos.keys())[0]
        native.alias(
            name = name + "_push",
            actual = ":{}_push_{}".format(name, back_compat_key),
            visibility = visibility,
        )
