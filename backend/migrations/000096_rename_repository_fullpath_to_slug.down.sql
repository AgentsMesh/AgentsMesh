-- Revert: repositories.slug → repositories.full_path
ALTER TABLE repositories RENAME COLUMN slug TO full_path;

DROP INDEX IF EXISTS idx_repositories_org_provider_slug;
CREATE UNIQUE INDEX idx_repositories_org_provider_fullpath
    ON repositories (organization_id, provider_type, provider_base_url, full_path)
    WHERE deleted_at IS NULL;
