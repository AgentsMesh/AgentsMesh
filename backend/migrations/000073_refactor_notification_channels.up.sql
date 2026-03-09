-- notification_preferences: toast/browser bool columns → channels JSONB
ALTER TABLE notification_preferences
    ADD COLUMN channels JSONB DEFAULT '{"toast":true,"browser":true}';

UPDATE notification_preferences
    SET channels = jsonb_build_object('toast', toast, 'browser', browser);

ALTER TABLE notification_preferences DROP COLUMN toast;
ALTER TABLE notification_preferences DROP COLUMN browser;

-- channels table: reserve delivery_config column for future channel-level delivery config
ALTER TABLE channels
    ADD COLUMN delivery_config JSONB;
