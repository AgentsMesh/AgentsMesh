-- Capture which agents still persist model/permission_mode on the legacy
-- pods.model and pods.permission_mode columns. New agents (codex-cli, aider,
-- gemini-cli, opencode, ...) rely on the AgentFile CONFIG snapshot at
-- pods.config_overrides exclusively; only the Claude family writes the legacy
-- columns for backward compatibility.
ALTER TABLE agents
  ADD COLUMN uses_legacy_columns BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE agents
   SET uses_legacy_columns = TRUE
 WHERE slug IN ('claude-code', 'claude');
