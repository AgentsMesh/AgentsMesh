-- Remove podfile_source from all agent types
UPDATE agent_types SET podfile_source = NULL;

-- Drop column
ALTER TABLE agent_types DROP COLUMN IF EXISTS podfile_source;
