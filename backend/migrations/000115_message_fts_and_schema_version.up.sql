-- Full-text search index on message body + schema versioning
CREATE INDEX idx_channel_messages_body_fts
ON channel_messages USING GIN (to_tsvector('english', body));

ALTER TABLE channel_messages ADD COLUMN IF NOT EXISTS schema_version INT NOT NULL DEFAULT 1;
