-- Phase 4+ cleanup: track ref mutations with an explicit timestamp so
-- audit / backlink UIs can sort by "last touched" and so updateRef is
-- distinguishable from addRef in time-travel queries.
--
-- Backfill existing rows with their created_at so the column is never NULL
-- before anyone updates the ref for the first time.

ALTER TABLE block_refs
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE block_refs SET updated_at = created_at WHERE updated_at IS NULL OR updated_at = NOW();
