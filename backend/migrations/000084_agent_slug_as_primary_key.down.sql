-- Reverse migration: restore numeric IDs
-- This is a destructive reverse — data mapping back to IDs is not trivially possible.
-- In practice, this migration should not be rolled back in production.

-- Restore agent_types id column (may already exist if 087 down ran first)
ALTER TABLE agent_types DROP CONSTRAINT IF EXISTS agent_types_pkey;
ALTER TABLE agent_types ADD COLUMN IF NOT EXISTS id SERIAL;
ALTER TABLE agent_types ADD PRIMARY KEY (id);
DO $$ BEGIN
  ALTER TABLE agent_types ADD CONSTRAINT agent_types_slug_key UNIQUE (slug);
EXCEPTION WHEN duplicate_table THEN NULL;
END $$;

-- Restore custom_agent_types id column
ALTER TABLE custom_agent_types DROP CONSTRAINT IF EXISTS custom_agent_types_pkey;
ALTER TABLE custom_agent_types ADD COLUMN IF NOT EXISTS id SERIAL;
ALTER TABLE custom_agent_types ADD PRIMARY KEY (id);
DO $$ BEGIN
  ALTER TABLE custom_agent_types ADD CONSTRAINT custom_agent_types_org_slug_key UNIQUE (organization_id, slug);
EXCEPTION WHEN duplicate_table THEN NULL;
END $$;

-- Restore agent_type_id columns (will be NULL — data lost)
ALTER TABLE pods ADD COLUMN IF NOT EXISTS agent_type_id INTEGER;
ALTER TABLE user_agent_configs ADD COLUMN IF NOT EXISTS agent_type_id INTEGER;
ALTER TABLE user_agent_credential_profiles ADD COLUMN IF NOT EXISTS agent_type_id INTEGER;
ALTER TABLE loops ADD COLUMN IF NOT EXISTS agent_type_id INTEGER;
