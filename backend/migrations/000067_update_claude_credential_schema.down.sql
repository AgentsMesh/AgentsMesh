-- Revert to original Claude Code credential schema (api_key only, required)
UPDATE agent_types
SET credential_schema = '[{"name":"api_key","type":"secret","env_var":"ANTHROPIC_API_KEY","required":true}]'
WHERE slug = 'claude-code';
