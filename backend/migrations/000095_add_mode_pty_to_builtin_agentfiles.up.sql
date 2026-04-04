-- Add MODE pty to all builtin agent AgentFiles that don't already have a MODE declaration.
-- This makes the default interaction mode explicit in the AgentFile source.
UPDATE agents
SET agentfile_source = 'MODE pty' || E'\n' || agentfile_source
WHERE slug IN ('claude-code', 'gemini-cli', 'codex-cli', 'aider', 'opencode')
  AND agentfile_source IS NOT NULL
  AND agentfile_source NOT LIKE '%MODE%';
