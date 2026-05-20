-- Phase 2: add slug column to api_keys. See 000136 for rationale.
ALTER TABLE api_keys ADD COLUMN slug VARCHAR(100);

ALTER TABLE api_keys ADD CONSTRAINT api_keys_slug_format
  CHECK (slug IS NULL OR (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100))
  NOT VALID;
