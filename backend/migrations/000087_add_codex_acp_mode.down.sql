-- Revert Codex CLI to PTY-only mode.
UPDATE agent_types SET supported_modes = 'pty' WHERE slug = 'codex-cli';
