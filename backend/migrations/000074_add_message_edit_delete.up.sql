ALTER TABLE channel_messages ADD COLUMN edited_at TIMESTAMPTZ;
ALTER TABLE channel_messages ADD COLUMN is_deleted BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_channel_messages_not_deleted ON channel_messages (channel_id, id)
    WHERE is_deleted = FALSE;
