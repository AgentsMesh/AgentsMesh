-- Remove MODE pty prefix from builtin agent PodFiles.
UPDATE agents
SET podfile_source = REPLACE(podfile_source, 'MODE pty' || E'\n', '')
WHERE slug IN ('claude-code', 'gemini-cli', 'codex-cli', 'aider', 'opencode')
  AND podfile_source IS NOT NULL
  AND podfile_source LIKE 'MODE pty%';
