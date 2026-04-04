-- Remove MODE pty prefix from builtin agent AgentFiles.
UPDATE agents
SET agentfile_source = REPLACE(agentfile_source, 'MODE pty' || E'\n', '')
WHERE slug IN ('claude-code', 'gemini-cli', 'codex-cli', 'aider', 'opencode')
  AND agentfile_source IS NOT NULL
  AND agentfile_source LIKE 'MODE pty%';
