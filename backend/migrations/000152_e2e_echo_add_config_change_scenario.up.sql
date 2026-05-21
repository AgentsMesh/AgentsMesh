-- 000152_e2e_echo_add_config_change_scenario.up.sql
-- Adds the `config_change_plan` value to the CONFIG scenario enum on the
-- e2e-echo AgentFile. Pairs with the runner-side ACPTransport extension
-- (session/control_request) and the mock-binary's handleControlRequest in
-- //runner/internal/agents/mockagent/acp_runtime.go.
--
-- Why a separate migration: 000151 is already in shipped state. SELECT
-- enum changes are agentfile_source rewrites; the safest pattern is a new
-- migration that only touches that source string, leaving 000151's content
-- alone for clean rollback semantics.

BEGIN;

UPDATE agents
SET agentfile_source = REPLACE(
    agentfile_source,
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit") = "echo"',
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit", "config_change_plan") = "echo"'
)
WHERE slug = 'e2e-echo';

COMMIT;
