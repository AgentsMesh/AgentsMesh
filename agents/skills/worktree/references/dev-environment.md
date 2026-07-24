# Worktree Development Environment

The canonical full-stack entrypoint is:

```bash
bazel run //deploy/dev:up
```

Use `//deploy/dev:backend_only` when the task does not need Web or Web Admin.
Other lifecycle targets are `//deploy/dev:clean`,
`//deploy/dev:reset_runners`, and `//deploy/dev:rebuild_runner`.

The environment is hybrid:

- Docker runs stateful infrastructure and containerized Runner dependencies.
- Backend, Relay, and the host Runner are launched through Bazel and ibazel.
- Web and Web Admin are Bazel Next.js development servers.
- Traefik routes the worktree's public HTTP, relay, and gRPC entrypoints.

Every worktree gets a generated `deploy/dev/.env` with isolated ports. After the
stack starts:

```bash
set -a
source deploy/dev/.env
set +a
```

Read values such as `HTTP_PORT`, `WEB_PORT`, `WEB_ADMIN_PORT`,
`POSTGRES_PORT`, `MINIO_API_PORT`, `MINIO_CONSOLE_PORT`, and
`TRAEFIK_DASHBOARD_PORT` from that file. Do not infer them from the main
checkout.

Useful logs:

```text
deploy/dev/runtime/backend/backend.log
deploy/dev/runtime/relay/relay.log
deploy/dev/runtime/runner/runner.log
deploy/dev/web.log
```

Only clean a stack that this workflow started and only after checking that no
parallel task depends on it.
