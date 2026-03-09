DROP INDEX IF EXISTS idx_channel_messages_not_deleted;
ALTER TABLE channel_messages DROP COLUMN IF EXISTS is_deleted;
ALTER TABLE channel_messages DROP COLUMN IF EXISTS edited_at;
