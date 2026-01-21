-- Add preparation_script and preparation_timeout columns to repositories table

-- Add columns to repositories table
ALTER TABLE repositories
ADD COLUMN IF NOT EXISTS preparation_script TEXT,
ADD COLUMN IF NOT EXISTS preparation_timeout INTEGER DEFAULT 300;

-- Add comment for documentation
COMMENT ON COLUMN repositories.preparation_script IS 'Script to run after worktree creation for workspace initialization';
COMMENT ON COLUMN repositories.preparation_timeout IS 'Timeout in seconds for preparation script execution (default 300)';
