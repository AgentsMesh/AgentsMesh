---
name: e2e
description: >-
  Selects and runs the appropriate AgentsMesh end-to-end suite for Web,
  Desktop, MCP, or iOS, including worktree-specific environment setup and
  browser-level verification. Use when a change needs E2E coverage, a user
  asks to execute or diagnose an E2E test, or a cross-service workflow must be
  verified against the real development stack.
---

# AgentsMesh E2E

Run the smallest real end-to-end test that proves the requested behavior, then
expand coverage in proportion to the change.

## Invariants

- Use the current worktree's generated `deploy/dev/.env`; never assume fixed
  ports or container names.
- A UI requirement must be verified by a browser or app UI test. Do not replace
  it with a raw API request.
- Start the development stack only when the selected suite needs it. Record
  whether this workflow started the stack before deciding whether to clean it.
- Authentication uses Connect-RPC. Never restore the removed REST login route;
  read `references/connect-rpc.md` when testing auth or proxy routing.
- Do not claim a suite passed unless its command completed successfully.
- Preserve test artifacts and relevant service logs when a failure occurs.

## Workflow

1. Inspect the requested behavior and changed files.
2. Select the suite from `references/test-matrix.md`. Read the selected target's
   BUILD file, config, and nearby tests before running it.
3. Reuse a healthy existing stack. Otherwise start `//deploy/dev:up` for Web
   flows or `//deploy/dev:backend_only` when frontends are not required.
4. Load `deploy/dev/.env` with automatic export enabled so Bazel can forward the
   worktree-specific values:

   ```bash
   set -a
   source deploy/dev/.env
   set +a
   ```

5. Run the narrowest relevant spec or test filter first. If it passes, run the
   owning target when the blast radius warrants it.
6. For UI behavior, inspect the rendered state and interaction result through
   the suite's browser/app assertions. Capture a screenshot or trace when the
   visual result is central to the bug.
7. On failure, report the failing assertion and inspect the applicable service
   logs and test artifacts listed in the test matrix.
8. Clean the stack with `bazel run //deploy/dev:clean` only when this workflow
   started it and no concurrent work depends on it.

## Result

Report the exact target and filter, actual worktree ports used, pass/fail
status, failing evidence when applicable, and any coverage that could not be
run with the reason.
