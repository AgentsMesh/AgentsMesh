-- Drop legacy config columns (AgentFile is now SSOT)
ALTER TABLE agents DROP COLUMN IF EXISTS config_schema;
ALTER TABLE agents DROP COLUMN IF EXISTS command_template;
ALTER TABLE agents DROP COLUMN IF EXISTS files_template;
ALTER TABLE agents DROP COLUMN IF EXISTS credential_schema;
ALTER TABLE agents DROP COLUMN IF EXISTS status_detection;

ALTER TABLE custom_agents DROP COLUMN IF EXISTS credential_schema;
ALTER TABLE custom_agents DROP COLUMN IF EXISTS status_detection;
