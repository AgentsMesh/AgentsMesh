-- 000152_e2e_echo_add_config_change_scenario.down.sql
-- Revert: remove config_change_plan from the scenario enum.

BEGIN;

UPDATE agents
SET agentfile_source = REPLACE(
    agentfile_source,
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit", "config_change_plan") = "echo"',
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit") = "echo"'
)
WHERE slug = 'e2e-echo';

COMMIT;
