ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_org_slug_unique;
ALTER TABLE channels ALTER COLUMN slug DROP NOT NULL;
ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_slug_format;
ALTER TABLE channels ADD CONSTRAINT channels_slug_format
  CHECK (slug IS NULL OR (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100))
  NOT VALID;
