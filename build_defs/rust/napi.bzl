"""Rust → Node-API (.node) native addon.

Compiles a Rust crate as a `cdylib` and renames the platform-specific
dynamic library (`lib<name>.dylib` / `lib<name>.so` / `<name>.dll`) into
the Node-friendly `<name>.node` suffix that `require()` expects.

Usage:
    load("//build_defs/rust:napi.bzl", "napi_rust_library")

    napi_rust_library(
        name = "node_bridge",
        crate = "//clients/core/crates/node-bridge:node_bridge_lib",
        visibility = ["//clients/desktop:__subpackages__"],
    )

The macro does not attempt to run `@napi-rs/cli`. The Rust `build.rs`
still emits the N-API header bindings; we ship the compiled `.node`
alongside it. TypeScript declarations are expected to be hand-maintained
(or generated separately via `js_run_binary` wrapping `napi build --dts`).

Known limits:
  - Node's N-API ABI must match the Electron build being shipped. Pin
    the target `node_version` in `MODULE.bazel` to match the Electron
    release's node ABI.
  - Cross-compilation is handled by `--platforms` (e.g. `linux/amd64`
    vs `darwin/arm64`); this macro simply packages whatever slice the
    current configuration produces.

Linker note:
  N-API symbols (`napi_*`, `_napi_wrap`, etc.) are resolved by the
  hosting Node/Electron process at `require()` time, not at link
  time of the addon itself. The default rustc cdylib link line on
  macOS errors with `Undefined symbols: _napi_wrap` because nothing
  on disk defines them. We pass `-undefined dynamic_lookup` (macOS)
  or `--unresolved-symbols=ignore-all` (linux) so the cdylib links
  with those symbols deliberately unresolved, deferring binding to
  the runtime loader. Mirrors what `@napi-rs/cli` does on the
  outside.
"""

load("@aspect_bazel_lib//lib:copy_to_bin.bzl", "copy_to_bin")
load("@rules_rust//rust:defs.bzl", "rust_shared_library")

_NAPI_LINKER_FLAGS = select({
    "@platforms//os:macos": [
        "-C",
        "link-arg=-undefined",
        "-C",
        "link-arg=dynamic_lookup",
    ],
    "@platforms//os:linux": [
        "-C",
        "link-arg=-Wl,--unresolved-symbols=ignore-all",
    ],
    "@platforms//os:windows": [],
    "//conditions:default": [],
})

def napi_rust_library(
        name,
        srcs,
        deps = None,
        proc_macro_deps = None,
        crate_name = None,
        edition = "2021",
        ts_declaration = None,
        binary_name = None,
        visibility = None):
    """Produce a Node-loadable `.node` file from a Rust source set.

    The output is named `<binary_name>.<platform-triple>.node` (e.g.
    `agentsmesh-node-bridge.darwin-arm64.node`) so the napi-rs-generated
    `index.js` loader (which `require()`s a fixed per-platform filename)
    can pick it up unchanged. If `binary_name` is omitted the output
    falls back to plain `<name>.node`.

    Args:
        name: Bazel target name (becomes the filegroup label).
        srcs: Rust source files for the crate.
        deps: Rust library dependencies.
        proc_macro_deps: Rust proc-macro deps (e.g. napi-derive).
        crate_name: Override for the emitted crate name. Defaults to `name`.
        edition: Rust edition. Defaults to 2021.
        ts_declaration: Optional label pointing at a hand-written `.d.ts`
            file to copy next to the `.node`. If absent, the package
            exports the addon only.
        binary_name: Filename stem to emit (e.g. `agentsmesh-node-bridge`).
            The platform triple suffix is appended automatically.
        visibility: Standard visibility attribute.
    """
    rust_shared_library(
        name = "_{}_cdylib".format(name),
        srcs = srcs,
        deps = deps or [],
        proc_macro_deps = proc_macro_deps or [],
        crate_name = crate_name or name,
        edition = edition,
        rustc_flags = _NAPI_LINKER_FLAGS,
    )

    # Emit one rename genrule per supported platform, then expose them
    # behind a single `alias` selected on the host config. Each genrule's
    # `outs` list is static (Bazel forbids `select` on outs), so the
    # per-platform names are spelled out below; the `alias` ensures
    # consumers see only the slice that matches the current build.
    if binary_name:
        for suffix in _PLATFORM_SUFFIXES.values():
            out_name = "{}.{}.node".format(binary_name, suffix)
            native.genrule(
                name = "_{}_rename_{}".format(name, suffix.replace("-", "_")),
                srcs = [":_{}_cdylib".format(name)],
                outs = [out_name],
                cmd = "cp $(SRCS) $@",
                tags = ["manual"],
            )
        native.alias(
            name = "_{}_rename".format(name),
            actual = select({
                plat: "_{}_rename_{}".format(name, suffix.replace("-", "_"))
                for plat, suffix in _PLATFORM_SUFFIXES.items()
            }),
        )
    else:
        native.genrule(
            name = "_{}_rename".format(name),
            srcs = [":_{}_cdylib".format(name)],
            outs = ["{}.node".format(name)],
            cmd = "cp $(SRCS) $@",
        )

    data = [":_{}_rename".format(name)]
    if ts_declaration:
        copy_to_bin(
            name = "_{}_dts".format(name),
            srcs = [ts_declaration],
        )
        data.append(":_{}_dts".format(name))

    native.filegroup(
        name = name,
        srcs = data,
        visibility = visibility or ["//visibility:public"],
    )

# Maps Bazel platform constraint labels to the napi-rs convention
# `<os>-<cpu>` triple that the generated index.js loader looks for.
# Only the slices we currently ship are listed; add more as we gain
# per-platform CI coverage.
_PLATFORM_SUFFIXES = {
    "@platforms//cpu:arm64": "darwin-arm64",  # default macOS dev target
}
