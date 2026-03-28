-- Re-add legacy config columns for rollback
ALTER TABLE agents ADD COLUMN config_schema JSONB NOT NULL DEFAULT '{}';
ALTER TABLE agents ADD COLUMN command_template JSONB NOT NULL DEFAULT '{}';
ALTER TABLE agents ADD COLUMN files_template JSONB;
ALTER TABLE agents ADD COLUMN credential_schema JSONB NOT NULL DEFAULT '[]';
ALTER TABLE agents ADD COLUMN status_detection JSONB;

ALTER TABLE custom_agents ADD COLUMN credential_schema JSONB NOT NULL DEFAULT '[]';
ALTER TABLE custom_agents ADD COLUMN status_detection JSONB;
