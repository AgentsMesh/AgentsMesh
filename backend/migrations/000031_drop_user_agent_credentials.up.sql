-- Drop legacy user_agent_credentials table
-- This table has been replaced by user_agent_credential_profiles which supports
-- multiple credential profiles per user/agent type combination.

DROP INDEX IF EXISTS idx_user_agent_creds_user;
DROP TABLE IF EXISTS user_agent_credentials;
