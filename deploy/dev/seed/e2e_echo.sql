-- deploy/dev/seed/e2e_echo.sql
--
-- e2e-echo mock agent — used by Playwright/MCP e2e tests to exercise the
-- full Backend → Runner gRPC → PTY pipeline + ACP JSON-RPC path without
-- depending on a real LLM CLI.
--
-- This file was extracted from migrations 000127/000150-000154 so that
-- production deployments (which apply migrations) never receive test
-- fixtures. Only dev/e2e environments invoke this seed via
-- deploy/dev/init-seed.sh.
--
-- The agent is marked is_internal=true so even if it does somehow end
-- up in a non-dev DB, the user-facing ListBuiltinActive view excludes it.
-- The runner discovery path (ListAllActive) still sees it, so e2e tests
-- can launch real pods against it.
--
-- Reference: //runner/internal/agents/mockagent/cmd/e2e-mock-agent
--            .claude/adr/2026-05-26-test-fixture-isolation.md

INSERT INTO agents (
    slug, name, description,
    launch_command, executable,
    is_builtin, is_active, is_internal,
    supported_modes, agentfile_source
) VALUES (
    'e2e-echo',
    'E2E Echo Agent',
    'Internal stub agent for end-to-end tests. Programmable PTY + ACP behavior via the e2e-mock-agent binary. Do not surface in production.',
    'bash',
    'bash',
    true,
    true,
    true,
    'pty,acp',
    E'# === Identity ===\nAGENT e2e-mock-agent\nEXECUTABLE e2e-mock-agent\n\n# === Mode ===\nMODE pty\nMODE acp "--mode=acp"\n\n# === Configuration ===\n# Scenario name maps to a behavior implemented by the mockagent PTY runtime or ACP registry.\nCONFIG scenario SELECT("echo", "terminal_render", "terminal_alt_snapshot", "streaming_3", "thinking_then_answer", "tool_call_edit", "permission_request_edit", "config_change_plan", "fail_after_1s", "malformed_json", "tool_call_failed", "log_warnings") = "echo"\n\n# === Capabilities ===\nMCP ON\n\n# === Build Logic ===\n# PTY mode uses the binary with no extra args (default mode=pty).\n# ACP mode is selected by the MODE acp "--mode=acp" line above.\n# The scenario flag applies to both modes.\narg "--scenario" config.scenario when config.scenario != ""\n'
)
ON CONFLICT (slug) DO UPDATE SET
    name             = EXCLUDED.name,
    description      = EXCLUDED.description,
    launch_command   = EXCLUDED.launch_command,
    executable       = EXCLUDED.executable,
    is_builtin       = EXCLUDED.is_builtin,
    is_active        = EXCLUDED.is_active,
    is_internal      = EXCLUDED.is_internal,
    supported_modes  = EXCLUDED.supported_modes,
    agentfile_source = EXCLUDED.agentfile_source,
    updated_at       = NOW();
