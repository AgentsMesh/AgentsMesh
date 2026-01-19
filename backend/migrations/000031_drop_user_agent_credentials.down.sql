-- Recreate legacy user_agent_credentials table (for rollback only)
-- Note: This table is deprecated and should not be used in new code.

CREATE TABLE user_agent_credentials (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_type_id BIGINT NOT NULL REFERENCES agent_types(id) ON DELETE CASCADE,

    credentials_encrypted JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, agent_type_id)
);

CREATE INDEX idx_user_agent_creds_user ON user_agent_credentials(user_id);
