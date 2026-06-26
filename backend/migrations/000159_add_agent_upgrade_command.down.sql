-- 000159_add_agent_upgrade_command.down.sql

ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_upgrade_pair_chk;
ALTER TABLE agents DROP COLUMN IF EXISTS upgrade_args;
ALTER TABLE agents DROP COLUMN IF EXISTS upgrade_manager;
