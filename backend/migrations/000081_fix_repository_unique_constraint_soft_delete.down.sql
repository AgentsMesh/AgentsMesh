-- Revert to plain unique constraint (loses soft-delete awareness).

DROP INDEX IF EXISTS repositories_org_provider_path_unique;

ALTER TABLE repositories
    ADD CONSTRAINT repositories_org_provider_path_unique
    UNIQUE(organization_id, provider_type, provider_base_url, full_path);
