-- User Agent Configs table
-- Stores user's personal agent runtime configurations (replacing organization-level configs)

CREATE TABLE IF NOT EXISTS user_agent_configs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_type_id BIGINT NOT NULL REFERENCES agent_types(id) ON DELETE CASCADE,
    config_values JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, agent_type_id)
);

-- Index for faster user config lookups
CREATE INDEX IF NOT EXISTS idx_user_agent_configs_user_id ON user_agent_configs(user_id);

-- Comment for documentation
COMMENT ON TABLE user_agent_configs IS 'Stores user personal agent runtime configurations';
COMMENT ON COLUMN user_agent_configs.config_values IS 'JSONB storing runtime config like model, permission_mode, think_level etc.';
