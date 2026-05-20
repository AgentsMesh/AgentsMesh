ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_org_slug_unique;
ALTER TABLE api_keys ALTER COLUMN slug DROP NOT NULL;
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_slug_format;
ALTER TABLE api_keys ADD CONSTRAINT api_keys_slug_format
  CHECK (slug IS NULL OR (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100))
  NOT VALID;
