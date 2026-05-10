# Contributing to AgentsMesh

Thanks for your interest in contributing.

## Before You Start

- Read the [README](./README.md) for project overview and local setup.
- By contributing, you agree that your contributions are licensed under this repository's [BSL 1.1](./LICENSE).

## Development Setup

1. Start local dependencies and seed data:

```bash
./deploy/dev/dev.sh
```

2. Start frontend locally (in a separate terminal):

```bash
cd web
pnpm install
pnpm dev
```

## Build and Test

Run relevant checks before opening a PR.

### Backend

```bash
bazel test //backend/...
bazel run //backend:lint
```

### Web

```bash
pnpm install --frozen-lockfile          # one-shot at repo root
bazel test //clients/web:lint
bazel build //clients/web:src           # tsc --noEmit (type check)
bazel test //clients/web:unit           # vitest
```

### Runner

```bash
bazel test //runner/...
bazel build //runner/cmd/runner:runner
bazel run //runner:lint
```

## Pull Request Guidelines

- Keep PRs focused and small where possible.
- Include context: what changed, why, and any tradeoffs.
- Add or update tests for behavior changes.
- Update docs when user-facing behavior or configuration changes.
- Ensure CI is green before requesting review.

## Commit Messages

Use clear, descriptive commit messages that explain intent.

## Reporting Bugs and Requesting Features

- Use GitHub Issues with the provided templates.
- For security-sensitive reports, see [SECURITY.md](./SECURITY.md).
