-- Phase 4 finale (channels): promote slug to NOT NULL + UNIQUE per-org +
-- VALIDATE the format CHECK. Deploy ONLY after the Phase 3 backfill program
-- has run with --apply for the channels.slug step and a fresh --dry-run
-- reports zero NULL rows. The old (organization_id, name) UNIQUE remains
-- as a defense-in-depth — both must be unique now.
ALTER TABLE channels VALIDATE CONSTRAINT channels_slug_format;
ALTER TABLE channels ALTER COLUMN slug SET NOT NULL;
ALTER TABLE channels ADD CONSTRAINT channels_org_slug_unique UNIQUE (organization_id, slug);
