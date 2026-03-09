-- Performance indexes for channel_messages cursor pagination and sender queries
CREATE INDEX IF NOT EXISTS idx_channel_messages_sender_user_id ON channel_messages(sender_user_id);
CREATE INDEX IF NOT EXISTS idx_channel_messages_cursor ON channel_messages(channel_id, id DESC);
