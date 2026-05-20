ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_slug_format;
ALTER TABLE agents ADD CONSTRAINT agents_slug_format
  CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100)
  NOT VALID;

ALTER TABLE custom_agents DROP CONSTRAINT IF EXISTS custom_agents_slug_format;
ALTER TABLE custom_agents ADD CONSTRAINT custom_agents_slug_format
  CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100)
  NOT VALID;
