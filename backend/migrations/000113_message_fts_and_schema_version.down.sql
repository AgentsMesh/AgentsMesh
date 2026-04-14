DROP INDEX IF EXISTS idx_channel_messages_body_fts;
ALTER TABLE channel_messages DROP COLUMN IF EXISTS schema_version;
