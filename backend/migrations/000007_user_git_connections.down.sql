-- Remove default git credential reference from users table
ALTER TABLE users DROP COLUMN IF EXISTS default_git_credential_id;

-- Drop user_git_connections table
DROP TRIGGER IF EXISTS update_user_git_connections_updated_at ON user_git_connections;
DROP TABLE IF EXISTS user_git_connections;
