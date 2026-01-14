-- User Git Connections
-- Stores manually added Git provider connections (PAT, SSH keys) for users
-- OAuth connections are stored in user_identities table

CREATE TABLE user_git_connections (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Provider info
    provider_type VARCHAR(50) NOT NULL,       -- github, gitlab, gitee, generic
    provider_name VARCHAR(100) NOT NULL,      -- User-defined name, e.g., "Company GitLab"
    base_url VARCHAR(255) NOT NULL,           -- https://gitlab.company.com

    -- Authentication
    auth_type VARCHAR(20) NOT NULL DEFAULT 'pat',  -- pat, ssh
    access_token_encrypted TEXT,              -- Encrypted PAT
    ssh_private_key_encrypted TEXT,           -- Encrypted SSH private key

    -- Provider user info (fetched during validation)
    external_user_id VARCHAR(255),            -- User ID on the platform
    external_username VARCHAR(255),           -- Username on the platform
    external_avatar_url TEXT,                 -- Avatar URL on the platform

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints: one connection per provider+base_url per user
    UNIQUE(user_id, provider_type, base_url)
);

CREATE INDEX idx_user_git_connections_user ON user_git_connections(user_id);
CREATE INDEX idx_user_git_connections_provider ON user_git_connections(provider_type, base_url);

-- Add trigger for updated_at
CREATE TRIGGER update_user_git_connections_updated_at
    BEFORE UPDATE ON user_git_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comment
COMMENT ON TABLE user_git_connections IS 'User-owned Git provider connections (PAT/SSH). OAuth connections are in user_identities.';

-- Add default git credential reference to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS default_git_credential_id BIGINT REFERENCES user_git_connections(id) ON DELETE SET NULL;
