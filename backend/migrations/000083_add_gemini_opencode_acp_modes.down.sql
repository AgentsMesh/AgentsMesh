-- Revert Gemini CLI and OpenCode to PTY-only
UPDATE agent_types SET supported_modes = 'pty' WHERE slug IN ('gemini-cli', 'opencode');
