-- Edit history table for channel messages
CREATE TABLE channel_message_edits (
    id               BIGSERIAL PRIMARY KEY,
    message_id       BIGINT NOT NULL REFERENCES channel_messages(id) ON DELETE CASCADE,
    editor_user_id   BIGINT REFERENCES users(id),
    editor_pod       VARCHAR(100),
    previous_body    TEXT NOT NULL,
    previous_content JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_message_edits_message ON channel_message_edits(message_id);
