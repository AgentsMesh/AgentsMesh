-- Fix: unique constraint must exclude soft-deleted rows so that
-- re-importing a previously deleted repository succeeds.
-- Replace the plain UNIQUE constraint with a partial unique index.

ALTER TABLE repositories
    DROP CONSTRAINT IF EXISTS repositories_org_provider_path_unique;

CREATE UNIQUE INDEX repositories_org_provider_path_unique
    ON repositories (organization_id, provider_type, provider_base_url, full_path)
    WHERE deleted_at IS NULL;
