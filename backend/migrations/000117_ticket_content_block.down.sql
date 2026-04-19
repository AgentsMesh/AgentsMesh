DROP INDEX IF EXISTS idx_tickets_content_block;
ALTER TABLE tickets DROP COLUMN IF EXISTS content_block_id;
