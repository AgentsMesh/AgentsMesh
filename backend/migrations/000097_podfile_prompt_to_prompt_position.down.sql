-- Revert: PROMPT_POSITION → prompt
UPDATE agents
SET podfile_source = REPLACE(podfile_source, 'PROMPT_POSITION prepend', 'prompt prepend'),
    updated_at = NOW()
WHERE is_builtin = true
  AND podfile_source LIKE '%PROMPT_POSITION prepend%';

UPDATE agents
SET podfile_source = REPLACE(podfile_source, 'PROMPT_POSITION append', 'prompt append'),
    updated_at = NOW()
WHERE is_builtin = true
  AND podfile_source LIKE '%PROMPT_POSITION append%';

UPDATE agents
SET podfile_source = REPLACE(podfile_source, 'PROMPT_POSITION none', 'prompt none'),
    updated_at = NOW()
WHERE is_builtin = true
  AND podfile_source LIKE '%PROMPT_POSITION none%';
