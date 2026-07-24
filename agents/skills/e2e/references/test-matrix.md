# AgentsMesh E2E Test Matrix

## Web

Use for browser-rendered Web behavior and Web-side API journeys.

```bash
bazel run //deploy/dev:up
set -a
source deploy/dev/.env
set +a
E2E_OUTPUT_BASE="${TMPDIR:-/tmp}/bazel-${COMPOSE_PROJECT_NAME}-web-e2e"
bazel --output_base="$E2E_OUTPUT_BASE" test \
  //clients/web/e2e-playwright:e2e \
  --test_tag_filters=e2e \
  --test_output=errors \
  --test_env=HTTP_PORT \
  --test_env=WEB_PORT \
  --test_env=POSTGRES_PORT \
  --test_env=COMPOSE_PROJECT_NAME
```

Pass a Playwright spec or title pattern after the target flags for a narrow
run:

```bash
--test_arg=tests/auth/login.spec.ts
--test_arg=--grep=<title-pattern>
```

Tests live under `clients/web/e2e-playwright/tests/`. Shared connection,
environment, database, and browser helpers live alongside them in `helpers/`,
`fixtures/`, and `pages/`. Install the browser once with `pnpm exec playwright
install chromium` if Playwright reports a missing executable.

## Desktop

Use for Electron main/preload/renderer integration, IPC contracts, persistence,
popout windows, or behavior that differs from the Web browser.

```bash
bazel run //deploy/dev:backend_only
set -a
source deploy/dev/.env
set +a
E2E_OUTPUT_BASE="${TMPDIR:-/tmp}/bazel-${COMPOSE_PROJECT_NAME}-desktop-e2e"
bazel --output_base="$E2E_OUTPUT_BASE" test \
  //clients/desktop:e2e \
  --test_tag_filters=e2e \
  --test_output=errors \
  --test_env=HTTP_PORT \
  --test_env=POSTGRES_PORT \
  --test_env=COMPOSE_PROJECT_NAME
```

Desktop specs live under `clients/desktop/e2e/tests/`. Pass a spec path with
`--test_arg` for a narrow run. Electron E2E requires a host capable of launching
the desktop application.

## MCP

Use for the Runner MCP HTTP server and the full Runner to Backend to database
chain.

```bash
bazel run //deploy/dev:backend_only
set -a
source deploy/dev/.env
set +a
export MCP_PORT="$RUNNER_MCP_PORT"
export POSTGRES_USER=agentsmesh
export POSTGRES_DB=agentsmesh
E2E_OUTPUT_BASE="${TMPDIR:-/tmp}/bazel-${COMPOSE_PROJECT_NAME}-mcp-e2e"
bazel --output_base="$E2E_OUTPUT_BASE" test \
  //tests/mcp-e2e/suites:e2e \
  --test_tag_filters=e2e \
  --test_output=errors \
  --test_env=MCP_PORT \
  --test_env=RUNNER_2_MCP_PORT \
  --test_env=BACKEND_HTTP_PORT \
  --test_env=POSTGRES_PORT \
  --test_env=POSTGRES_USER \
  --test_env=POSTGRES_PASSWORD \
  --test_env=POSTGRES_DB
```

Use `--test_filter` for a narrow Go test. Treat `.github/workflows/bazel.yml`
as the authoritative environment contract when older prose disagrees with CI.

The dedicated output bases keep each test command from tearing down the Bazel
processes that serve the running development stack.

## iOS

Use for SwiftUI navigation, native rendering, and iOS-specific integration on a
macOS host with Xcode and a booted simulator:

```bash
bazel test //clients/ios/Tests/AgentsMeshUITests:AgentsMesh_e2e
```

Specs live under `clients/ios/Tests/AgentsMeshUITests/Specs/`. Read
`Helpers/Env.swift` before overriding the development API endpoint.

## Failure Evidence

```text
deploy/dev/runtime/backend/backend.log
deploy/dev/runtime/relay/relay.log
deploy/dev/runtime/runner/runner.log
deploy/dev/web.log
test-results/
clients/web/e2e-playwright/playwright-report/
clients/desktop/playwright-report/
```
