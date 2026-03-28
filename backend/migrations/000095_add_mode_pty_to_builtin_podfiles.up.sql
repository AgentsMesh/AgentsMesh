-- Add MODE pty to all builtin agent PodFiles that don't already have a MODE declaration.
-- This makes the default interaction mode explicit in the PodFile source.
UPDATE agents
SET podfile_source = 'MODE pty' || E'\n' || podfile_source
WHERE slug IN ('claude-code', 'gemini-cli', 'codex-cli', 'aider', 'opencode')
  AND podfile_source IS NOT NULL
  AND podfile_source NOT LIKE '%MODE%';
