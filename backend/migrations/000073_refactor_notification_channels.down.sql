ALTER TABLE channels DROP COLUMN IF EXISTS delivery_config;

ALTER TABLE notification_preferences ADD COLUMN toast BOOLEAN DEFAULT TRUE;
ALTER TABLE notification_preferences ADD COLUMN browser BOOLEAN DEFAULT TRUE;

UPDATE notification_preferences SET
    toast = COALESCE((channels->>'toast')::boolean, true),
    browser = COALESCE((channels->>'browser')::boolean, true);

ALTER TABLE notification_preferences DROP COLUMN channels;
