-- Add supported_modes for Gemini CLI and OpenCode (native ACP support)
UPDATE agent_types SET supported_modes = 'pty,acp' WHERE slug IN ('gemini-cli', 'opencode');
