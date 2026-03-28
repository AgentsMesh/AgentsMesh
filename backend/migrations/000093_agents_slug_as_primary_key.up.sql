-- Change agents table PK from id (BIGSERIAL) to slug (VARCHAR)
-- The id column is no longer referenced anywhere after migration 084.

-- Drop foreign key constraints that reference the old agents.id PK
-- These tables were added by main branch and reference agent_types_pkey
ALTER TABLE organization_agents DROP CONSTRAINT IF EXISTS organization_agents_agent_type_id_fkey;
ALTER TABLE organization_agent_configs DROP CONSTRAINT IF EXISTS organization_agent_configs_agent_type_id_fkey;

-- Drop the old BIGSERIAL primary key
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agent_types_pkey;
ALTER TABLE agents DROP COLUMN IF EXISTS id;

-- Make slug the primary key (it already has a UNIQUE constraint)
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agent_types_slug_key;
ALTER TABLE agents ADD PRIMARY KEY (slug);

-- Recreate foreign keys pointing to slug instead of id
-- organization_agents: change agent_type_id → agent_slug reference
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'organization_agents' AND column_name = 'agent_type_id') THEN
    ALTER TABLE organization_agents DROP COLUMN agent_type_id;
    ALTER TABLE organization_agents ADD COLUMN IF NOT EXISTS agent_slug VARCHAR(100) REFERENCES agents(slug);
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'organization_agent_configs' AND column_name = 'agent_type_id') THEN
    ALTER TABLE organization_agent_configs DROP COLUMN agent_type_id;
    ALTER TABLE organization_agent_configs ADD COLUMN IF NOT EXISTS agent_slug VARCHAR(100) REFERENCES agents(slug);
  END IF;
END $$;

-- Same for custom_agents
ALTER TABLE custom_agents DROP CONSTRAINT IF EXISTS custom_agent_types_pkey;
ALTER TABLE custom_agents DROP COLUMN IF EXISTS id;

-- custom_agents: composite PK (organization_id, slug)
ALTER TABLE custom_agents DROP CONSTRAINT IF EXISTS custom_agent_types_org_slug_key;
ALTER TABLE custom_agents ADD PRIMARY KEY (organization_id, slug);
