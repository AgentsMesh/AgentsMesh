-- 000153_e2e_echo_add_error_scenarios.down.sql

BEGIN;

UPDATE agents
SET agentfile_source = REPLACE(
    agentfile_source,
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit", "config_change_plan", "fail_after_1s", "malformed_json", "tool_call_failed", "log_warnings") = "echo"',
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit", "config_change_plan") = "echo"'
)
WHERE slug = 'e2e-echo';

COMMIT;
