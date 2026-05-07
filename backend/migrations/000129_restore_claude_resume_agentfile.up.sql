-- Restore Claude Code session arg wiring in AgentFile after command_template removal.
--
-- On Pod creation the backend mints a uuid into pods.session_id and injects it
-- as config.session_id via systemOverrides. The AgentFile must pass that uuid
-- to the CLI with `--session-id` so Claude Code writes its transcript under the
-- same id the backend tracks. Without it, Claude self-generates a different id
-- and later `--resume <backend_uuid>` fails with "No conversation found with
-- session ID: ...".
--
-- This migration injects two mutually exclusive Build Logic lines right after
-- the `--model` arg:
--   * --session-id  — on new sessions (when session_id is set and not resuming)
--   * --resume      — when resuming (resume_enabled=true, resume_session=<id>)
-- `not config.resume_enabled` keeps the two args from ever appearing together.
UPDATE agents
SET agentfile_source = REPLACE(
    agentfile_source,
    E'# === Build Logic ===\narg "--model" config.model when config.model != ""\n\n',
    E'# === Build Logic ===\narg "--model" config.model when config.model != ""\narg "--session-id" config.session_id when config.session_id != "" and not config.resume_enabled\narg "--resume" config.resume_session when config.resume_enabled\n\n'
)
WHERE slug = 'claude-code'
  AND agentfile_source NOT LIKE '%arg "--session-id" config.session_id%'
  AND agentfile_source NOT LIKE '%arg "--resume" config.resume_session%';
