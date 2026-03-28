-- Revert: restore id BIGSERIAL PK for agents and custom_agents

-- agents: restore id column and PK
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_pkey;
ALTER TABLE agents ADD COLUMN id BIGSERIAL;
ALTER TABLE agents ADD PRIMARY KEY (id);
ALTER TABLE agents ADD CONSTRAINT agents_slug_key UNIQUE (slug);

-- custom_agents: restore id column and PK
ALTER TABLE custom_agents DROP CONSTRAINT IF EXISTS custom_agents_pkey;
ALTER TABLE custom_agents ADD COLUMN id BIGSERIAL PRIMARY KEY;
ALTER TABLE custom_agents ADD CONSTRAINT custom_agents_org_slug_key UNIQUE (organization_id, slug);
