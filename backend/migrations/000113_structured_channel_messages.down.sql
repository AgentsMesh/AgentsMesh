-- Revert to legacy channel_messages schema
DROP TABLE IF EXISTS channel_messages CASCADE;

CREATE TABLE channel_messages (
    id              BIGSERIAL PRIMARY KEY,
    channel_id      BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sender_pod      VARCHAR(100),
    sender_user_id  BIGINT REFERENCES users(id),
    message_type    VARCHAR(50) NOT NULL DEFAULT 'text',
    content         TEXT NOT NULL,
    metadata        JSONB,
    edited_at       TIMESTAMPTZ,
    is_deleted      BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_channel_messages_cursor       ON channel_messages(channel_id, id DESC);
CREATE INDEX idx_channel_messages_sender_pod   ON channel_messages(sender_pod)      WHERE sender_pod IS NOT NULL;
CREATE INDEX idx_channel_messages_sender_user  ON channel_messages(sender_user_id)  WHERE sender_user_id IS NOT NULL;
CREATE INDEX idx_channel_messages_not_deleted  ON channel_messages(channel_id, id)  WHERE is_deleted = FALSE;
