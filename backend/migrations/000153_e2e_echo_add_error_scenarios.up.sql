-- 000153_e2e_echo_add_error_scenarios.up.sql
-- Appends defensive-path scenarios to the e2e-echo CONFIG scenario enum:
--   fail_after_1s     agent process exits mid-session (OnExit, pod stopped)
--   malformed_json    non-JSON output interleaved with valid messages
--   tool_call_failed  tool ends with status=failed + errorMessage
--   log_warnings      stderr warn/error → UI LogEntry surface
--
-- Pairs with //runner/internal/agents/mockagent/scenario_errors.go and
-- clients/web/e2e-playwright/tests/pod/acp-ui-errors.spec.ts.

BEGIN;

UPDATE agents
SET agentfile_source = REPLACE(
    agentfile_source,
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit", "config_change_plan") = "echo"',
    'CONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit", "config_change_plan", "fail_after_1s", "malformed_json", "tool_call_failed", "log_warnings") = "echo"'
)
WHERE slug = 'e2e-echo';

COMMIT;
