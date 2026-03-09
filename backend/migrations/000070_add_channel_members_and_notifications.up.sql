-- Channel members: tracks which users are part of a channel
CREATE TABLE channel_members (
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_muted   BOOLEAN DEFAULT FALSE,
    joined_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_members_user_id ON channel_members(user_id);

-- Channel read states: tracks last read position per user per channel
CREATE TABLE channel_read_states (
    channel_id           BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id BIGINT,
    last_read_at         TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (channel_id, user_id)
);

-- Notification preferences: per-user, per-source notification settings
CREATE TABLE notification_preferences (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source     VARCHAR(50) NOT NULL,
    entity_id  VARCHAR(200) NOT NULL DEFAULT '',
    is_muted   BOOLEAN DEFAULT FALSE,
    toast      BOOLEAN DEFAULT TRUE,
    browser    BOOLEAN DEFAULT TRUE,
    UNIQUE (user_id, source, entity_id)
);

CREATE INDEX idx_notification_preferences_user_source ON notification_preferences(user_id, source);
