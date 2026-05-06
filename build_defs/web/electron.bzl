"""Electron desktop app packaging for AgentsMesh.

Wraps the electron-vite build pipeline (main + preload + renderer) and
electron-builder packaging into Bazel actions:

  - `electron_vite_build` runs `electron-vite build` over the package
    sources and produces the `out/` tree artifact (main + preload +
    renderer bundles) under `bazel-bin/`.
  - `electron_builder_app` consumes that tree plus the thin-shell
    `package.json` and an electron-builder config, runs
    `electron-builder` from a Bazel sandbox, and emits a `dist/` tree
    artifact containing platform installers (`.dmg`, `.exe`,
    `.AppImage`, etc.).

Usage:
    load("//build_defs/web:electron.bzl",
         "electron_vite_build", "electron_builder_app")

    electron_vite_build(
        name = "out",
        srcs = glob(["src/**/*"]) + ["electron.vite.config.ts", "tsconfig.json"],
    )

    electron_builder_app(
        name = "dist",
        out = ":out",
        package_json = "package.json",
        config = "electron-builder.yml",
    )
"""

load("@aspect_bazel_lib//lib:copy_to_directory.bzl", "copy_to_directory")
load("@aspect_rules_js//js:defs.bzl", "js_run_binary")
load("@npm//:electron-builder/package_json.bzl", electron_builder_bin = "bin")
load("@npm//:electron-vite/package_json.bzl", electron_vite_bin = "bin")
load(":internal_package.bzl", "generated_package_json")

def electron_vite_build(
        name,
        srcs,
        out_dir = "out",
        chdir = None,
        visibility = None,
        **kwargs):
    """Run `electron-vite build` over the package as a Bazel action.

    Produces a tree artifact (`out_dir`) containing:
        main/index.js          — main-process bundle
        preload/index.js       — preload bundle
        renderer/              — renderer assets

    Args:
        name: Target name. Output tree label is `:<name>`.
        srcs: All source files electron-vite needs (TS sources +
            config + tsconfig). Pass via `glob(["src/**/*", ...])`.
            Must include `electron.vite.config.ts` — electron-vite
            picks it up by convention from the package root.
        out_dir: Output directory name (default `out`). Bazel tree
            artifact created relative to bazel-bin/<package>/.
        chdir: CWD electron-vite runs in. Defaults to the calling
            package.
        visibility: Standard visibility.
        **kwargs: Forwarded to `js_run_binary`.
    """
    electron_vite_bin.electron_vite_binary(
        name = name + "_bin",
        visibility = ["//visibility:private"],
    )

    js_run_binary(
        name = name,
        srcs = srcs + ["//:node_modules"],
        args = ["build"],
        chdir = chdir or native.package_name(),
        out_dirs = [out_dir],
        tool = ":" + name + "_bin",
        visibility = visibility,
        **kwargs
    )

def electron_builder_app(
        name,
        out,
        package_name,
        version = "0.1.0",
        electron_main = "./out/main/index.js",
        config = "electron-builder.yml",
        extra_srcs = None,
        dependencies = None,
        out_dir = "dist",
        platform = None,
        chdir = None,
        visibility = None,
        tags = None,
        **kwargs):
    """Run `electron-builder` over a pre-built `out/` tree as a Bazel action.

    Wraps the electron-builder CLI: stages the `out/` tree (produced by
    `electron_vite_build`) plus a synthesised `package.json` and an
    electron-builder config file into a sandbox CWD, then runs
    `electron-builder` to produce platform-specific installers
    (`.dmg`, `.exe`, `.AppImage`, etc.) into a `dist_dir` tree artifact.

    The `package.json` is generated from `package_name` / `version` /
    `electron_main` — no source-tree thin-shell required. electron-
    builder reads `name`/`version` for installer naming; Electron itself
    reads `main` to find the bundled entry point.

    Args:
        name: Target name. Output tree label is `:<name>` (the `dist/`
            tree).
        out: Label of the `electron_vite_build` target whose tree
            artifact contains `main/` + `preload/` + `renderer/`.
        package_name: npm-style name for the synthesised package.json
            (e.g. `desktop`). electron-builder uses it for installer
            file names.
        version: Semver string written into package.json's `version`.
            Used by electron-builder for output naming.
        electron_main: Path inside the staged tree to the main-process
            entry. Defaults to `./out/main/index.js` (matches what
            `electron_vite_build` produces).
        dependencies: Dict of runtime-required npm package names → version
            specs (use `*` for first-party). electron-builder walks this
            to decide which `node_modules/<pkg>/` subtrees ship inside
            `app.asar`. Native modules listed in electron-vite's
            `rollupOptions.external` MUST appear here, otherwise main-
            process `require('<pkg>')` throws "Cannot find module" at
            runtime — the file is in node_modules on disk but never
            made it into the packaged app.
        config: electron-builder config filename. Default
            `electron-builder.yml`.
        extra_srcs: Additional sources to ship into the staging
            sandbox (build/, icons, entitlements, etc.).
        out_dir: Output directory name (default `dist`). Tree
            artifact created relative to bazel-bin/<package>/.
        platform: Platform flag — one of `mac`, `win`, `linux`. If
            unset, electron-builder builds for the host platform.
        chdir: CWD electron-builder runs in. Defaults to the calling
            package.
        visibility: Standard visibility.
        tags: Bazel tags. Defaults to `["manual", "electron_builder",
            "no-sandbox", "local"]` because cross-platform packaging
            requires the matching host (macOS for `.dmg`, etc.) and
            calls native binaries (hdiutil/nsis/AppImage) the sandbox
            can't accommodate.
        **kwargs: Forwarded to `js_run_binary`.
    """
    electron_builder_bin.electron_builder_binary(
        name = name + "_bin",
        # Critical: disable aspect_rules_js's `node-patches/fs.cjs`
        # interception. electron-builder's `appFileCopier.walk` uses
        # `lstat`+`readlink` recursively over the staging tree; the
        # patches' multi-hop readlink walker throws EINVAL on regular
        # files inside bazel-out (the `package.json` symlink chain is
        # deeper than node-patches expects). Unpatched fs lets
        # electron-builder walk normally.
        patch_node_fs = False,
        visibility = ["//visibility:private"],
    )

    # Synthesise package.json from BUILD attrs — no source-tree thin
    # shell. The output is `bazel-bin/<pkg>/package.json` named via
    # `:<name>_pkg_json`; copy_to_directory below pulls it into staging.
    pkg_json_target = name + "_pkg_json"
    generated_package_json(
        name = pkg_json_target,
        package_name = package_name,
        version = version,
        main = electron_main,
        dependencies = dependencies,
        extra = {
            "description": "AgentsMesh Desktop — Electron-hosted client.",
            # electron-builder's AppImage target rejects a package.json
            # without `homepage` (Linux Build :dist fails fatally; mac
            # dmg / win nsis are silent on the same field but read it for
            # installer metadata). Synthesised here so a single source-
            # tree config covers all three platforms.
            "homepage": "https://agentsmesh.ai",
            # `author` (with `email`) is also strictly enforced on the
            # Linux side — `electron-builder` reads it for AppImage
            # X-Apparmor-Profile + .deb maintainer fields. Missing it
            # blows up exactly like the missing `homepage` did. mac dmg
            # / win nsis are tolerant but pull `author.name` into
            # CFBundleDisplayName / installer Author metadata when set.
            "author": {
                "name": "AgentsMesh",
                "email": "support@agentsmesh.ai",
            },
        },
    )

    # Stage `:out` + generated package.json + config into a hermetic
    # directory of real files (not aspect_rules_js symlink chains).
    # electron-builder's `appFileCopier.walk` uses `lstat` + `readlink`
    # recursively; the multi-hop pnpm/aspect symlinks tree under
    # bazel-out trips its symlink walker (EINVAL on readlink). The
    # staging copy produces a `.app/` layout electron-builder can walk
    # without hitting node-patches/fs.cjs.
    staging_name = name + "_staging"
    copy_to_directory(
        name = staging_name,
        srcs = [out, ":" + pkg_json_target, config] + (extra_srcs or []),
        # `out` carries `out/` prefix from electron_vite_build's
        # tree-artifact root; package.json + config land at top-level.
        root_paths = [native.package_name()],
        # Mirror real file content so electron-builder's readlink
        # walker stops following hop chains.
        replace_prefixes = {},
        hardlink = "off",
        allow_overwrites = True,
        verbose = False,
    )

    args = [
        "--config",
        config,
        "--projectDir",
        staging_name,
    ]
    if platform == "mac":
        args.append("--mac")
    elif platform == "win":
        args.append("--win")
    elif platform == "linux":
        args.append("--linux")

    srcs = [
        ":" + staging_name,
        "//:node_modules",
    ]

    # First-party deps declared in synth package.json must be reachable
    # under the staging tree's node_modules walk, otherwise electron-
    # builder's app-builder skips them (they're not in the pnpm workspace
    # closure that `//:node_modules` materialises). For each entry in
    # `dependencies`, attach the matching `//:node_modules/<pkg>` target
    # so Bazel stages the linked package alongside the action.
    #
    # Without this the builder produces a smaller asar (~13MB vs ~37MB)
    # with no app.asar.unpacked tree, the .node binary never lands in
    # Resources/, and the packaged app throws "Cannot find module" at
    # runtime — invisible to a clean build that re-uses cached actions
    # from an earlier state where the link still happened to exist.
    for pkg_name in (dependencies or {}).keys():
        srcs.append("//:node_modules/" + pkg_name)

    # Default tags include `no-sandbox`/`local` because macOS dmg creation
    # invokes `hdiutil`, which needs real filesystem operations and a
    # `/private` mount the darwin-sandbox forbids. Linux AppImage
    # creation also calls native binaries with similar constraints.
    # Bazel still tracks inputs/outputs hermetically — sandbox-off only
    # changes the *execution* sandbox, not the action graph.
    default_tags = ["manual", "electron_builder", "no-sandbox", "local"]

    js_run_binary(
        name = name,
        srcs = srcs,
        args = args,
        chdir = chdir or native.package_name(),
        out_dirs = [out_dir],
        tool = ":" + name + "_bin",
        tags = tags if tags != None else default_tags,
        visibility = visibility,
        # Disable aspect_rules_js's `node-patches/fs.cjs` interception
        # — see binary above for the rationale (electron-builder's
        # recursive lstat/readlink walk hits EINVAL on bazel-out
        # symlink chains the patches were never designed to follow).
        patch_node_fs = False,
        # electron-builder writes tons of cache state (electron binary
        # download, app-builder downloads, etc.) under $HOME — without
        # an isolated CACHE dir, the sandbox bombs out trying to create
        # `/.cache`. Point its caches at $TMPDIR.
        env = {
            "ELECTRON_BUILDER_CACHE": "$$(pwd)/.electron-builder-cache",
            "ELECTRON_CACHE": "$$(pwd)/.electron-cache",
        },
        **kwargs
    )
