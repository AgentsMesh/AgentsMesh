-- Rename repositories.full_path → repositories.slug
-- full_path has always served as the repository slug (e.g., "dev-org/demo-api").
-- Aligning the column name with other entities (agents, tickets) that use "slug".

-- 1. Rename column
ALTER TABLE repositories RENAME COLUMN full_path TO slug;

-- 2. Drop old unique index and create new one with renamed column
DROP INDEX IF EXISTS idx_repositories_org_provider_fullpath;
CREATE UNIQUE INDEX idx_repositories_org_provider_slug
    ON repositories (organization_id, provider_type, provider_base_url, slug)
    WHERE deleted_at IS NULL;
