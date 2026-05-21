-- 000151_e2e_echo_dual_mode.up.sql
-- Upgrade e2e-echo from a single bash echo loop into a programmable, dual-mode
-- (PTY + ACP) mock agent driven by the new e2e-mock-agent binary.
--
-- Phase 1 of the broader "e2e-echo as universal agent test harness" plan:
--   * PTY behavior preserved bit-for-bit (ready signal, env dump for the
--     EnvBundle e2e, line-echo loop). Bash is replaced by e2e-mock-agent --mode=pty.
--   * NEW: MODE acp uses the same binary with --mode=acp, speaking JSON-RPC 2.0.
--     The default `echo` scenario echoes the prompt back as a single
--     `agent_message_chunk`, mirroring PTY semantics so symmetric assertions
--     are possible across modes.
--   * The runner (or its sandbox) must expose the e2e-mock-agent binary on
--     PATH. The Bazel target is //runner/internal/agents/mockagent/cmd/e2e-mock-agent.
--
-- Future phases will extend CONFIG scenario with streaming, tool_call,
-- permission_request, config_change, etc.

BEGIN;

UPDATE agents
SET agentfile_source = E'# === Identity ===\nAGENT e2e-mock-agent\nEXECUTABLE e2e-mock-agent\n\n# === Mode ===\nMODE pty\nMODE acp "--mode=acp"\n\n# === Configuration ===\n# Scenario name maps to a behavior registered in the mockagent package.\n# Keep this enum in sync with //runner/internal/agents/mockagent/scenarios.go.\nCONFIG scenario SELECT("echo", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit") = "echo"\n\n# === Capabilities ===\nMCP ON\n\n# === Build Logic ===\n# PTY mode uses the binary with no extra args (default mode=pty).\n# ACP mode is selected by the MODE acp "--mode=acp" line above.\n# The scenario flag applies to both modes.\narg "--scenario" config.scenario when config.scenario != ""\n'
WHERE slug = 'e2e-echo';

COMMIT;
