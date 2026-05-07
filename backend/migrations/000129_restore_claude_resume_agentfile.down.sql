UPDATE agents
SET agentfile_source = REPLACE(
    agentfile_source,
    E'# === Build Logic ===\narg "--model" config.model when config.model != ""\narg "--session-id" config.session_id when config.session_id != "" and not config.resume_enabled\narg "--resume" config.resume_session when config.resume_enabled\n\n',
    E'# === Build Logic ===\narg "--model" config.model when config.model != ""\n\n'
)
WHERE slug = 'claude-code';
