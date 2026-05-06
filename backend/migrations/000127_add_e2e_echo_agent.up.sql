-- Add e2e-echo agent for MCP end-to-end testing.
-- This agent is a stub: it doesn't connect to any LLM API, just runs a bash
-- echo loop. Used by tests/mcp-e2e/ to spin up real Pods through the full
-- backend → runner gRPC → PTY pipeline so MCP tool calls can be validated
-- end-to-end without consuming model credits or external API quotas.
--
-- The agent is registered globally (is_builtin=true). Frontends that don't
-- want users to see it can filter by slug prefix or by is_internal once that
-- column is added; for now the description makes the intent clear.
INSERT INTO agents (slug, name, description, launch_command, executable, is_builtin, is_active, supported_modes, agentfile_source)
VALUES (
    'e2e-echo',
    'E2E Echo Agent',
    'Internal stub agent for MCP end-to-end tests. Reads from stdin and echoes "got: <line>". Do not use in production.',
    'bash',
    'bash',
    true,
    true,
    'pty',
    E'AGENT bash\nEXECUTABLE bash\n\nMODE pty\nMCP ON\n\n# Build logic: a single -c argument that runs an echo loop.\n# - Prints "ready" once on startup so tests can detect liveness via\n#   get_pod_snapshot.\n# - Reads stdin in a loop and echoes "got: <line>" so send_pod_input +\n#   get_pod_snapshot can be tested for round-trip.\n# Note: AGENT decl is the runner-side launch command (resolved through PATH),\n# not the agent slug. The slug "e2e-echo" lives only in the agents table.\narg "-c" "echo ready; while IFS= read -r line; do echo \\"got: $line\\"; done"\n'
)
ON CONFLICT (slug) DO NOTHING;
