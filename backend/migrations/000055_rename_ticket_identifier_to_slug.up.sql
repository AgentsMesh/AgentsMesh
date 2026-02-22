-- Rename column (metadata-only, instant, no table rewrite)
ALTER TABLE tickets RENAME COLUMN identifier TO slug;

-- Rename index to match
ALTER INDEX idx_tickets_org_identifier RENAME TO idx_tickets_org_slug;
