-- Migrate AgentFile syntax: `prompt prepend/append/none` → `PROMPT_POSITION prepend/append/none`
-- Part of AgentFile SSOT refactoring: PROMPT is now a declaration for content,
-- PROMPT_POSITION is a separate declaration for position.

UPDATE agents
SET agentfile_source = REPLACE(agentfile_source, 'prompt prepend', 'PROMPT_POSITION prepend'),
    updated_at = NOW()
WHERE is_builtin = true
  AND agentfile_source LIKE '%prompt prepend%';

UPDATE agents
SET agentfile_source = REPLACE(agentfile_source, 'prompt append', 'PROMPT_POSITION append'),
    updated_at = NOW()
WHERE is_builtin = true
  AND agentfile_source LIKE '%prompt append%';

UPDATE agents
SET agentfile_source = REPLACE(agentfile_source, 'prompt none', 'PROMPT_POSITION none'),
    updated_at = NOW()
WHERE is_builtin = true
  AND agentfile_source LIKE '%prompt none%';
