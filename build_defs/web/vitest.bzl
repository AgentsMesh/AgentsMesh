"""Vitest test runner wrapped as a Bazel test.

Vitest's package.json doesn't expose the CLI in its `exports` map, so
we can't use the aspect_rules_js auto-generated `bin.vitest_test` —
unlike Playwright. Instead we drive Vitest through a CJS shim that
`createRequire(CWD).resolve('vitest/package.json')` the pnpm-linked
package and spawns the bin file directly.
"""

load("@aspect_rules_js//js:defs.bzl", "js_test")

def vitest_test(
        name,
        srcs,
        config = "vitest.config.ts",
        data = None,
        deps = None,
        env = None,
        size = "medium",
        timeout = None,
        tags = None,
        chdir = None,
        extra_args = None):
    """Run `vitest run` against a set of sources as a Bazel test.

    Args:
        name: Target name.
        srcs: Test files (`*.test.ts`, `*.test.tsx`).
        config: Vitest config file (default `vitest.config.ts`).
        data: Extra runfiles (fixtures, configs, html templates).
        deps: JS library deps.
        env: Environment variables forwarded to the test process.
        size: Bazel test size.
        timeout: Optional timeout string.
        tags: Bazel tags forwarded verbatim.
        chdir: Working directory (defaults to package path).
        extra_args: Optional Vitest CLI arguments, for example coverage flags.
    """
    kwargs = {}
    if timeout:
        kwargs["timeout"] = timeout
    merged_env = dict(env or {})
    merged_env.setdefault("VITEST", "true")
    js_test(
        name = name,
        entry_point = "//build_defs/web:vitest_shim",
        args = [
            "run",
            "--config",
            config,
            "--reporter=default",
            "--no-color",
        ] + (extra_args or []),
        data = (data or []) + srcs + [config] + (deps or []) + [
            "//:node_modules/vitest",
        ],
        env = merged_env,
        chdir = chdir or native.package_name(),
        size = size,
        tags = tags or [],
        **kwargs
    )
