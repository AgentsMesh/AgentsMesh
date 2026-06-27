-- 000161_channel_read_states_manually_unread.down.sql
BEGIN;

ALTER TABLE channel_read_states DROP COLUMN IF EXISTS manually_unread;

COMMIT;
