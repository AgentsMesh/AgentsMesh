-- agent_type_id → agent_slug 全局迁移
-- 全栈统一用 slug 标识 Agent

-- ============================================================
-- Step 1: 为引用表添加 agent_slug 列并填充数据
-- ============================================================

ALTER TABLE pods ADD COLUMN IF NOT EXISTS agent_slug VARCHAR(100);
UPDATE pods SET agent_slug = (SELECT slug FROM agent_types WHERE id = pods.agent_type_id) WHERE agent_type_id IS NOT NULL AND agent_slug IS NULL;
UPDATE pods SET agent_slug = (SELECT slug FROM custom_agent_types WHERE id = pods.custom_agent_type_id) WHERE custom_agent_type_id IS NOT NULL AND agent_slug IS NULL;

ALTER TABLE user_agent_configs ADD COLUMN IF NOT EXISTS agent_slug VARCHAR(100);
UPDATE user_agent_configs SET agent_slug = (SELECT slug FROM agent_types WHERE id = user_agent_configs.agent_type_id) WHERE agent_slug IS NULL;

ALTER TABLE user_agent_credential_profiles ADD COLUMN IF NOT EXISTS agent_slug VARCHAR(100);
UPDATE user_agent_credential_profiles SET agent_slug = (SELECT slug FROM agent_types WHERE id = user_agent_credential_profiles.agent_type_id) WHERE agent_slug IS NULL;

ALTER TABLE loops ADD COLUMN IF NOT EXISTS agent_slug VARCHAR(100);
UPDATE loops SET agent_slug = (SELECT slug FROM agent_types WHERE id = loops.agent_type_id) WHERE agent_type_id IS NOT NULL AND agent_slug IS NULL;
UPDATE loops SET agent_slug = (SELECT slug FROM custom_agent_types WHERE id = loops.custom_agent_type_id) WHERE custom_agent_type_id IS NOT NULL AND agent_slug IS NULL;

-- ============================================================
-- Step 2: 删除旧的数字 ID 列
-- ============================================================

ALTER TABLE pods DROP COLUMN IF EXISTS agent_type_id;
ALTER TABLE pods DROP COLUMN IF EXISTS custom_agent_type_id;

ALTER TABLE user_agent_configs DROP COLUMN IF EXISTS agent_type_id;

ALTER TABLE user_agent_credential_profiles DROP COLUMN IF EXISTS agent_type_id;

ALTER TABLE loops DROP COLUMN IF EXISTS agent_type_id;
ALTER TABLE loops DROP COLUMN IF EXISTS custom_agent_type_id;

-- ============================================================
-- Step 3: 创建索引
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_pods_agent_slug ON pods(agent_slug);
CREATE INDEX IF NOT EXISTS idx_user_agent_configs_agent_slug ON user_agent_configs(agent_slug);
CREATE INDEX IF NOT EXISTS idx_user_agent_credential_profiles_agent_slug ON user_agent_credential_profiles(agent_slug);
CREATE INDEX IF NOT EXISTS idx_loops_agent_slug ON loops(agent_slug);
