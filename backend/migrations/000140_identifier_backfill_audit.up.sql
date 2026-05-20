-- Phase 3: audit trail for the one-off identifier backfill program at
-- backend/cmd/backfill-identifiers. Each rewritten cell appends a row here so
-- on-call can reconstruct the (old, new) mapping for any user whose
-- @mention or URL just changed.
CREATE TABLE IF NOT EXISTS identifier_backfill_audit (
    id BIGSERIAL PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    column_name VARCHAR(50) NOT NULL,
    row_id BIGINT NOT NULL,
    old_value TEXT NOT NULL,
    new_value TEXT NOT NULL,
    ran_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_identifier_backfill_audit_table_row
  ON identifier_backfill_audit(table_name, row_id);
