ALTER TABLE tickets RENAME COLUMN slug TO identifier;

ALTER INDEX idx_tickets_org_slug RENAME TO idx_tickets_org_identifier;
