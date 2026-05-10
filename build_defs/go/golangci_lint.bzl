"""golangci_lint — run hermetic golangci-lint over a Go subtree.

Wraps `@multitool//tools/golangci-lint` in a `sh_binary` invokable via
`bazel run //backend:lint`. CI calls the same command instead of
`golangci-lint-action@v9`, so the linter version + flags + exclusion
rules live in `multitool.lock.json` + `<subtree>/.golangci.yml` —
hermetic on every developer / CI machine.

Why a binary, not a test:
- golangci-lint walks the full Go module dep graph (GOMODCACHE,
  vendored deps, generated code). The Bazel test sandbox strips
  symlinks Go's loader needs, and re-materializing the entire dep
  graph as runfiles per subtree is wasteful.
- `bazel run` exposes `BUILD_WORKSPACE_DIRECTORY`, so the lint runs
  against the live source tree just like the previous
  `golangci-lint-action`. Exit code propagates naturally → CI fails
  on lint errors.

The repo is a single Go module rooted at `//:go.mod`. The runner script
cds into the workspace root, sets `GOWORK=off`, and feeds golangci-lint
the subtree-scoped pattern `./<subtree>/...`. Per-subtree `.golangci.yml`
files keep lint rule sets independent (e.g. backend allows gorm-style
global state, runner doesn't).

Usage (from `backend/BUILD.bazel`):

    load("//build_defs/go:golangci_lint.bzl", "golangci_lint")

    golangci_lint(
        name = "lint",
        config = ".golangci.yml",
    )

Then `bazel run //backend:lint` runs golangci-lint over the entire
`backend/` subtree. No args needed; pass extra flags after `--`:

    bazel run //backend:lint -- --fix
"""

def golangci_lint(
        name,
        config,
        module_dir = None,
        **kwargs):
    """Run golangci-lint over a Go subtree of the root module.

    Args:
        name: Binary target name. Convention: `lint`.
        config: Path to the subtree's `.golangci.yml`, package-relative.
        module_dir: Workspace-relative dir to scope the lint to. The
            runner cds to the workspace root and runs
            `golangci-lint run ./<module_dir>/...`. Defaults to the
            calling package (the common case — top-level subtree
            BUILD.bazel like //backend, //runner, //relay).
        **kwargs: Forwarded to `native.sh_binary` (e.g., `tags`,
            `visibility`).
    """
    if module_dir == None:
        module_dir = native.package_name()

    native.sh_binary(
        name = name,
        srcs = ["//build_defs/go:golangci_lint_runner.sh"],
        data = [
            config,
            "@multitool//tools/golangci-lint",
        ],
        args = [
            "$(rlocationpath @multitool//tools/golangci-lint)",
            module_dir,
            "$(rlocationpath :%s)" % config,
        ],
        deps = ["@bazel_tools//tools/bash/runfiles"],
        **kwargs
    )
