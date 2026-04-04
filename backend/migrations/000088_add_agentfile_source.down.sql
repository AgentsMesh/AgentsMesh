-- Remove agentfile_source from all agent types
UPDATE agent_types SET agentfile_source = NULL;

-- Drop column
ALTER TABLE agent_types DROP COLUMN IF EXISTS agentfile_source;
