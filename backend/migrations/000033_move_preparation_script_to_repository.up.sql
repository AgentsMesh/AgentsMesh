-- Move preparation_script and preparation_timeout from user_agentpod_settings to repositories

-- Step 1: Add columns to repositories table
ALTER TABLE repositories
ADD COLUMN preparation_script TEXT,
ADD COLUMN preparation_timeout INTEGER DEFAULT 300;

-- Step 2: Remove columns from user_agentpod_settings table
ALTER TABLE user_agentpod_settings
DROP COLUMN IF EXISTS preparation_script,
DROP COLUMN IF EXISTS preparation_timeout;

-- Add comment for documentation
COMMENT ON COLUMN repositories.preparation_script IS 'Script to run after worktree creation for workspace initialization';
COMMENT ON COLUMN repositories.preparation_timeout IS 'Timeout in seconds for preparation script execution (default 300)';
