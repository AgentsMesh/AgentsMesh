# AgentsMesh Runner — dev image
#
# This image is the AI CLI overlay on top of the bazel-built runner
# binary. The binary itself is produced by `bazel build
# //runner/cmd/runner:runner` (cross-compiled to linux/amd64 by
# dev.sh on macOS hosts) and copied in via the build context. There's
# no Go toolchain or `air` inside the image — the binary is the unit
# of deployment, same compile path as CI / production.
#
# Hot reload is intentionally off: the runner spawns AI agent
# processes that need a sandbox boundary, so we trade rebuild speed
# for isolation. To rebuild after a runner code change:
#   ./dev.sh --rebuild-runner
FROM debian:trixie-slim

# Base packages + AI CLI dependencies (Node 20+ for Gemini CLI).
RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates tzdata bash curl wget openssh-client openssl \
    python3 build-essential g++ make sudo jq file \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# AI CLIs — Claude Code / OpenAI Codex / Gemini CLI. Pinned to whatever
# npm latest publishes; bump explicitly via `docker compose build runner`.
RUN npm install -g \
        @anthropic-ai/claude-code \
        @openai/codex \
        @google/gemini-cli \
    && npm cache clean --force

# Non-root user the runner runs as. Owns ~/ for AI CLI configs and
# /workspace for pod scratch space. Pre-create the runtime dirs so a
# fresh named volume for ~/.agentsmesh/ has runner-writable ownership.
RUN useradd --create-home --uid 1000 --shell /bin/bash runner \
    && mkdir -p /workspace /app /home/runner/.agentsmesh /home/runner/.claude /home/runner/.codex /home/runner/.gemini \
    && chown -R runner:runner /workspace /app /home/runner \
    && echo 'runner ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/runner

# Bazel-built runner binary — cross-compiled for linux/amd64 by dev.sh
# (see build_runner_binary). dev.sh copies the binary out of bazel-bin
# (which is a symlink that docker build context can't follow across
# the symlink boundary) to deploy/dev/runner-binary, and the build
# context is `deploy/dev`.
COPY --chmod=0755 runner-binary /usr/local/bin/agentsmesh-runner

# Bazel-built e2e-mock-agent — drives the `e2e-echo` AgentFile in both PTY
# and ACP modes. Required for mcp-e2e / envbundle-e2e / acp-ui-e2e suites
# that create pods without depending on real LLM CLIs. See
# //runner/internal/agents/mockagent for the implementation, and
# build_mock_agent_binary in lib/host_services.sh for the cross-compile.
COPY --chmod=0755 e2e-mock-agent-binary /usr/local/bin/e2e-mock-agent

# Cross-compiled real Loopal (linux/amd64) for hands-on Loopal control-console
# experience — `EXECUTABLE loopal` in the loopal builtin AgentFile resolves to
# this. Built locally via `bazel build //:loopal --config=linux-x86` in the
# Loopal repo, copied to deploy/dev/loopal-binary.
COPY --chmod=0755 loopal-binary /usr/local/bin/loopal

USER runner
WORKDIR /app
ENTRYPOINT ["/usr/local/bin/runner-entrypoint.sh"]
