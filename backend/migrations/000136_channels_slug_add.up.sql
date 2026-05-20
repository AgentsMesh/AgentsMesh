-- Phase 2: add slug column to channels alongside existing UNIQUE name. The
-- column is nullable + no UNIQUE here so old rows pass and new writers can
-- backfill incrementally via channel_registry. Phase 4 promotes to NOT NULL
-- + UNIQUE (organization_id, slug) after backfill completes.
ALTER TABLE channels ADD COLUMN slug VARCHAR(100);

ALTER TABLE channels ADD CONSTRAINT channels_slug_format
  CHECK (slug IS NULL OR (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100))
  NOT VALID;
