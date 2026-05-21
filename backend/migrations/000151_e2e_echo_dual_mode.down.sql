-- 000151_e2e_echo_dual_mode.down.sql
-- Revert to the bash-based single-mode (PTY) e2e-echo agent from migration 000150.

BEGIN;

UPDATE agents
SET agentfile_source = E'AGENT bash\nEXECUTABLE bash\n\nMODE pty\nMCP ON\n\n# Build logic: dump whitelisted env to disk + run the echo loop.\n# - Prints "ready" first (mcp-e2e liveness anchor).\n# - Dumps env entries matching E2E_TEST_, ANTHROPIC_, CLAUDE_ prefixes to\n#   /tmp/e2e-echo-env-dump-$$ so the EnvBundle e2e can verify injection\n#   via `docker exec` without depending on PTY streaming.\n# - Reads stdin and echoes "got: <line>" (existing round-trip behavior).\narg "-c" "echo ready; env | grep -E ''^(E2E_TEST_|ANTHROPIC_|CLAUDE_)'' | sort > /tmp/e2e-echo-env-dump-$$; while IFS= read -r line; do echo \\"got: $line\\"; done"\n'
WHERE slug = 'e2e-echo';

COMMIT;
