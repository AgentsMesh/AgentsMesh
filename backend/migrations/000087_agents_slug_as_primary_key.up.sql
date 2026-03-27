-- Change agents table PK from id (BIGSERIAL) to slug (VARCHAR)
-- The id column is no longer referenced anywhere after migration 084.

-- Drop the old BIGSERIAL primary key
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agent_types_pkey;
ALTER TABLE agents DROP COLUMN IF EXISTS id;

-- Make slug the primary key (it already has a UNIQUE constraint)
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agent_types_slug_key;
ALTER TABLE agents ADD PRIMARY KEY (slug);

-- Same for custom_agents
ALTER TABLE custom_agents DROP CONSTRAINT IF EXISTS custom_agent_types_pkey;
ALTER TABLE custom_agents DROP COLUMN IF EXISTS id;

-- custom_agents: composite PK (organization_id, slug)
ALTER TABLE custom_agents DROP CONSTRAINT IF EXISTS custom_agent_types_org_slug_key;
ALTER TABLE custom_agents ADD PRIMARY KEY (organization_id, slug);
