-- Revert: PROMPT_POSITION → prompt
UPDATE agents
SET agentfile_source = REPLACE(agentfile_source, 'PROMPT_POSITION prepend', 'prompt prepend'),
    updated_at = NOW()
WHERE is_builtin = true
  AND agentfile_source LIKE '%PROMPT_POSITION prepend%';

UPDATE agents
SET agentfile_source = REPLACE(agentfile_source, 'PROMPT_POSITION append', 'prompt append'),
    updated_at = NOW()
WHERE is_builtin = true
  AND agentfile_source LIKE '%PROMPT_POSITION append%';

UPDATE agents
SET agentfile_source = REPLACE(agentfile_source, 'PROMPT_POSITION none', 'prompt none'),
    updated_at = NOW()
WHERE is_builtin = true
  AND agentfile_source LIKE '%PROMPT_POSITION none%';
