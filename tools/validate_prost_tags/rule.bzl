"""Bazel rule wrapping tools/validate_prost_tags/validate_prost_tags.py.

Per `proto_library` target with a paired Rust file, add:

    load("//tools/validate_prost_tags:rule.bzl", "validate_prost_tags")

    validate_prost_tags(
        name = "extension_validate",
        proto = "skill_registry.proto",
        rust = "//clients/core/crates/types:src/extension_proto.rs",
    )

The rule is a `bazel test` target: `bazel test //path:extension_validate`
runs the validator and surfaces drift errors with the file pair that
caused them. Pass means proto and Rust agree on every field-name↔tag pair
for every struct present in both.

We intentionally model this as a test target (not a `genrule` that fails
at build time) so a transient drift during a refactor doesn't block
`bazel build`; the test target is added to the per-service CI matrix in
.github/workflows.
"""

def _validate_prost_tags_test_impl(ctx):
    test_script = ctx.actions.declare_file(ctx.label.name + ".sh")
    # Bazel test runfiles: scripts run from `<bin>.runfiles/_main/`, the
    # validator's short_path is relative to that root, so prepending it to
    # the runfiles_dir gives the absolute path we exec.
    ctx.actions.write(
        output = test_script,
        content = """#!/usr/bin/env bash
set -euo pipefail
runfiles_root="${{TEST_SRCDIR}}/${{TEST_WORKSPACE:-_main}}"
exec "$runfiles_root/{validator}" "$runfiles_root/{proto}" "$runfiles_root/{rust}"
""".format(
            validator = ctx.executable._validator.short_path,
            proto = ctx.file.proto.short_path,
            rust = ctx.file.rust.short_path,
        ),
        is_executable = True,
    )
    runfiles = ctx.runfiles(files = [
        ctx.file.proto,
        ctx.file.rust,
        ctx.executable._validator,
    ])
    # Merge the validator's own runfiles (the .py data dep) so the wrapped
    # sh_binary can find it.
    runfiles = runfiles.merge(ctx.attr._validator[DefaultInfo].default_runfiles)
    return [DefaultInfo(executable = test_script, runfiles = runfiles)]

# Rule name MUST end with `_test` — Bazel enforces this for `test = True`
# rules. Caller-facing macro is `validate_prost_tags` below; it forwards to
# this rule with the test suffix appended.
_validate_prost_tags_test = rule(
    implementation = _validate_prost_tags_test_impl,
    test = True,
    attrs = {
        "proto": attr.label(allow_single_file = [".proto"], mandatory = True),
        "rust": attr.label(allow_single_file = [".rs"], mandatory = True),
        "_validator": attr.label(
            default = "//tools/validate_prost_tags:validate_prost_tags",
            executable = True,
            cfg = "exec",
        ),
    },
)

def validate_prost_tags(name, proto, rust, **kwargs):
    """Macro wrapping `_validate_prost_tags_test` so callers spell the rule
    in its natural form (`validate_prost_tags(...)`) without dealing with
    Bazel's "_test" suffix requirement on test rules. The underlying test
    target is `<name>` itself — `bazel test //...:<name>` runs it."""
    _validate_prost_tags_test(name = name, proto = proto, rust = rust, **kwargs)
