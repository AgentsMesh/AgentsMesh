-- Rollback: Move preparation_script and preparation_timeout back to user_agentpod_settings

-- Step 1: Add columns back to user_agentpod_settings table
ALTER TABLE user_agentpod_settings
ADD COLUMN preparation_script TEXT,
ADD COLUMN preparation_timeout INTEGER NOT NULL DEFAULT 300;

-- Step 2: Remove columns from repositories table
ALTER TABLE repositories
DROP COLUMN IF EXISTS preparation_script,
DROP COLUMN IF EXISTS preparation_timeout;
