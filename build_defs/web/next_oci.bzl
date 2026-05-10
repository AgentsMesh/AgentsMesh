"""Package a Next.js standalone build into a distroless OCI image.

Consumes `:<name>_server_dir` from `next_standalone_app` (in
//build_defs/web:next.bzl) — a flat directory tree containing
server.js + .next/static/ + public/ + NFT-traced node_modules. We
pkg_tar that under /app, drop in `entrypoint.mjs` + `healthcheck.mjs`
from the calling package, and launch via the distroless base's node
binary.

Distinct from the legacy `js_image_layer`-based path because:
  - Standalone output is self-contained: server.js needs only `node`
    and the NFT-traced files, no bash launcher / runfiles tree.
  - The base is real distroless (gcr.io/distroless/nodejs20-debian12),
    which has no /bin/sh — `js_image_layer` would not run there.

Usage:
    load("//build_defs/web:next_oci.bzl", "next_oci_image")

    next_oci_image(
        name = "image",
        server_dir = ":next_server_dir",
        extra_files = ["entrypoint.mjs", "healthcheck.mjs"],
        app_dir = "/app/clients/web",
        env = {"PORT": "3000"},
        exposed_ports = ["3000/tcp"],
        repositories = {
            "agentsmesh": "registry.agentsmesh.ai/agentsmesh/web",
            "dockerhub": "agentsmesh/web",
        },
    )
"""

load("@rules_oci//oci:defs.bzl", "oci_image", "oci_load")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")
load("//build_defs/docker:oci_push.bzl", "oci_push_targets")

def next_oci_image(
        name,
        server_dir,
        extra_files = [],
        app_dir = None,
        base = "@distroless_nodejs",
        env = {},
        exposed_ports = ["3000/tcp"],
        labels = {},
        repositories = {},
        repository = None,
        visibility = ["//visibility:public"]):
    """Distroless OCI image around a Next.js standalone server tree.

    Args:
        name: target name.
        server_dir: Label of `:<n>_server_dir` from `next_standalone_app`.
        extra_files: list of file Labels dropped verbatim under /app
            (typically `entrypoint.mjs` and `healthcheck.mjs`).
        app_dir: absolute path inside the container that contains
            `server.js` (e.g. `/app/clients/web`). Baked into the
            `APP_DIR` env so `entrypoint.mjs` knows where to chdir.
            Defaults to `/app` (server.js at root — only used by callers
            that flatten the standalone tree themselves).
        base: base image; defaults to `@distroless_nodejs`.
        env: extra env vars (merged after `APP_DIR`).
        exposed_ports: container ports.
        labels: OCI labels.
        repositories: dict mapping registry key → full repo URL.
        repository: back-compat single-registry form.
        visibility: standard visibility.
    """

    # The standalone server tree is named `app` (set via `copy_to_directory`'s
    # `out = "app"` in next.bzl), so the pkg_tar in-archive path before
    # any strip is `<package>/app/<contents>`. We strip the workspace path
    # to land everything at `/app/<contents>` in the image.
    #
    # `owner = "65532.65532"` makes everything writable by the distroless
    # `:nonroot` user, which entrypoint.mjs needs in order to substitute
    # placeholders inside `.next/`. Default mode 0644 keeps file
    # contents owner-writable + world-readable.
    pkg_tar(
        name = name + "_app_layer",
        srcs = [server_dir],
        package_dir = "/",
        strip_prefix = "/" + native.package_name(),
        owner = "65532.65532",
        # pkg_tar defaults to mode 0555 (read+exec only). Override to
        # 0644 so the nonroot user can rewrite `.next/*.js` placeholder
        # values at container boot. server.js itself stays runnable
        # (Node only needs read+exec to load the file as JS).
        mode = "0644",
    )

    tars = [":" + name + "_app_layer"]
    if extra_files:
        pkg_tar(
            name = name + "_runtime_layer",
            srcs = extra_files,
            package_dir = "/app",
            owner = "65532.65532",
            mode = "0755",
        )
        tars.append(":" + name + "_runtime_layer")

    image_env = {}
    if app_dir:
        image_env["APP_DIR"] = app_dir
    image_env.update(env)

    oci_image(
        name = name,
        base = base,
        # distroless/nodejs20-debian12 ships node at /nodejs/bin/node and
        # sets it as the default ENTRYPOINT. We override to invoke
        # entrypoint.mjs first (runtime placeholder substitution +
        # signal-forwarding spawn of server.js).
        entrypoint = ["/nodejs/bin/node", "/app/entrypoint.mjs"],
        env = image_env,
        exposed_ports = exposed_ports,
        labels = labels,
        tars = tars,
        visibility = visibility,
        workdir = "/app",
    )

    oci_load(
        name = name + "_tarball",
        image = ":" + name,
        repo_tags = ["agentsmesh/" + name + ":latest"],
        visibility = visibility,
    )

    oci_push_targets(
        name = name,
        image = ":" + name,
        repositories = repositories,
        repository = repository,
        visibility = visibility,
    )
